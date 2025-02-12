package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoTagParser holds parsed values from the `dynamo` struct tag.
type DynamoTagParser struct {
	AttributeName string
	KeyType       string
	Index         string
	Required      bool
}

// TableNamer should be implemented by items which specify their own table name.
type TableNamer interface {
	TableName() string
}

// ParseDynamoTag parses a struct field tag and returns a parser.
func ParseDynamoTag(tag string) *DynamoTagParser {
	parser := &DynamoTagParser{}
	if tag == "" {
		return parser
	}
	options := strings.Split(tag, ",")
	if len(options) > 0 && options[0] != "" {
		parser.AttributeName = options[0]
	}
	for _, opt := range options[1:] {
		switch {
		case strings.HasPrefix(opt, "key="):
			parser.KeyType = strings.TrimPrefix(opt, "key=")
		case strings.HasPrefix(opt, "index="):
			parser.Index = strings.TrimPrefix(opt, "index=")
		case opt == "required":
			parser.Required = true
		}
	}
	return parser
}

// DynamoDBClient is used by the repository to interface with DynamoDB.
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

// Repository implements basic CRUD operations using DynamoDB.
type Repository struct {
	client    DynamoDBClient
	tableName string
}

var (
	ErrDuplicateKey = errors.New("item with this key already exists")
	ErrNotFound     = errors.New("item not found")
)

// NewRepository returns a new Repository.
func NewRepository(client DynamoDBClient, defaultTableName string) *Repository {
	return &Repository{
		client:    client,
		tableName: defaultTableName,
	}
}

// getTableName returns the table name for an item. It checks if the item implements TableNamer.
func (r *Repository) getTableName(item interface{}) string {
	if tableNamer, ok := item.(TableNamer); ok {
		return tableNamer.TableName()
	}
	if reflect.ValueOf(item).Kind() == reflect.Ptr {
		if tableNamer, ok := reflect.ValueOf(item).Elem().Interface().(TableNamer); ok {
			return tableNamer.TableName()
		}
	}
	return r.tableName
}

// Create stores an item in DynamoDB.
func (r *Repository) Create(ctx context.Context, item interface{}) error {
	if err := validateStruct(item); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}
	tableName := r.getTableName(item)
	conditionExpression := createConditionExpression(item)
	input := &dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                av,
		ConditionExpression: aws.String(conditionExpression),
	}
	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("failed to put item: %w", err)
	}
	return nil
}

// validateStruct ensures that required fields (as specified by the `dynamo` tag) are non-zero.
func validateStruct(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("item must be a struct")
	}
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag, ok := field.Tag.Lookup("dynamo"); ok {
			parser := ParseDynamoTag(tag)
			if parser.Required {
				fieldValue := val.Field(i)
				if fieldValue.IsZero() {
					return fmt.Errorf("field %s is required", field.Name)
				}
			}
		}
	}
	return nil
}

// createConditionExpression builds a condition expression that checks for non-existence of key attributes.
// This is used during Create to protect against duplicate keys.
func createConditionExpression(v interface{}) string {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	conditions := []string{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag, ok := field.Tag.Lookup("dynamo"); ok {
			parser := ParseDynamoTag(tag)
			if parser.KeyType == "hash" {
				conditions = append(conditions, fmt.Sprintf("attribute_not_exists(%s)", parser.AttributeName))
			}
		}
	}
	if len(conditions) == 0 {
		return ""
	}
	return strings.Join(conditions, " AND ")
}

// FindByID retrieves an item by its key.
func (r *Repository) FindByID(ctx context.Context, id interface{}, out interface{}) error {
	tableName := r.getTableName(out)
	elemType := reflect.TypeOf(out)
	if elemType.Kind() != reflect.Ptr {
		return fmt.Errorf("out must be a pointer")
	}
	elemType = elemType.Elem()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("out must be a pointer to struct")
	}
	var keyAttribute string
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if tag, ok := field.Tag.Lookup("dynamo"); ok {
			parser := ParseDynamoTag(tag)
			if parser.KeyType == "hash" {
				keyAttribute = parser.AttributeName
				break
			}
		}
	}
	if keyAttribute == "" {
		return fmt.Errorf("no hash key defined in struct")
	}
	av, err := attributevalue.Marshal(id)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}
	key := map[string]types.AttributeValue{
		keyAttribute: av,
	}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	}
	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if result.Item == nil {
		return fmt.Errorf("item not found")
	}
	err = attributevalue.UnmarshalMap(result.Item, out)
	if err != nil {
		return fmt.Errorf("failed to unmarshal item: %w", err)
	}
	return nil
}

// FindByParameter retrieves items by a given parameter value.
// It uses a Query if an index exists for the parameter
// and a Scan otherwise.
func (r *Repository) FindByParameter(ctx context.Context, parameter string, value interface{}, out interface{}) error {
	outType := reflect.TypeOf(out)
	if outType.Kind() != reflect.Ptr {
		return fmt.Errorf("out must be a pointer to slice")
	}
	sliceType := outType.Elem()
	if sliceType.Kind() != reflect.Slice {
		return fmt.Errorf("out must be a pointer to slice")
	}
	elemType := sliceType.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("slice element must be a struct")
	}
	tableName := r.getTableName(reflect.New(elemType).Interface())
	var useQuery bool
	var indexName string
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if tag, ok := field.Tag.Lookup("dynamo"); ok {
			parser := ParseDynamoTag(tag)
			if parser.AttributeName == parameter && parser.Index != "" {
				useQuery = true
				indexName = parser.Index
				break
			}
		}
	}
	marshaledValue, err := attributevalue.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	exprAttrValues := map[string]types.AttributeValue{
		":v": marshaledValue,
	}
	if useQuery {
		input := &dynamodb.QueryInput{
			TableName:                 aws.String(tableName),
			IndexName:                 aws.String(indexName),
			KeyConditionExpression:    aws.String(fmt.Sprintf("%s = :v", parameter)),
			ExpressionAttributeValues: exprAttrValues,
		}
		result, err := r.client.Query(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to query: %w", err)
		}
		err = attributevalue.UnmarshalListOfMaps(result.Items, out)
		if err != nil {
			return fmt.Errorf("failed to unmarshal query result: %w", err)
		}
	} else {
		input := &dynamodb.ScanInput{
			TableName:                 aws.String(tableName),
			FilterExpression:          aws.String(fmt.Sprintf("%s = :v", parameter)),
			ExpressionAttributeValues: exprAttrValues,
		}
		result, err := r.client.Scan(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to scan: %w", err)
		}
		err = attributevalue.UnmarshalListOfMaps(result.Items, out)
		if err != nil {
			return fmt.Errorf("failed to unmarshal scan result: %w", err)
		}
	}
	return nil
}

// GetAll retrieves all items from a table.
func (r *Repository) GetAll(ctx context.Context, out interface{}) error {
	outType := reflect.TypeOf(out)
	if outType.Kind() != reflect.Ptr {
		return fmt.Errorf("out must be a pointer to slice")
	}
	sliceType := outType.Elem()
	if sliceType.Kind() != reflect.Slice {
		return fmt.Errorf("out must be a pointer to slice")
	}
	elemType := sliceType.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("slice element must be a struct")
	}
	tableName := r.getTableName(reflect.New(elemType).Interface())
	input := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to scan: %w", err)
	}
	err = attributevalue.UnmarshalListOfMaps(result.Items, out)
	if err != nil {
		return fmt.Errorf("failed to unmarshal scan result: %w", err)
	}
	return nil
}

// Update updates an existing item in DynamoDB. It:
//  1. Uses reflection to extract the key field (the one with `key=hash`).
//  2. Builds an UpdateExpression with all other fields tagged with `dynamo`.
//  3. Uses a ConditionExpression to ensure the item exists.
//
// If no updatable field is found or if the key is missing the update will return an error.
func (r *Repository) Update(ctx context.Context, item interface{}) error {
	// Get the underlying struct value.
	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("item must be a struct")
	}
	tableName := r.getTableName(item)

	var keyAttr string
	keyMap := make(map[string]types.AttributeValue)
	updateExpressions := []string{}
	exprAttrNames := make(map[string]string)
	exprAttrValues := make(map[string]types.AttributeValue)

	typ := val.Type()
	// Walk through all struct fields.
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)
		tag, ok := field.Tag.Lookup("dynamo")
		if !ok {
			continue
		}
		parser := ParseDynamoTag(tag)
		if parser.AttributeName == "" {
			continue
		}
		if parser.KeyType == "hash" {
			// Mark this as the key field.
			keyAttr = parser.AttributeName
			marshaledVal, err := attributevalue.Marshal(fieldValue.Interface())
			if err != nil {
				return fmt.Errorf("failed to marshal key field %s: %w", field.Name, err)
			}
			keyMap[keyAttr] = marshaledVal
		} else {
			// Build update expression part for non-key fields.
			placeholderName := "#" + parser.AttributeName
			placeholderValue := ":" + parser.AttributeName
			updateExpressions = append(updateExpressions, fmt.Sprintf("%s = %s", placeholderName, placeholderValue))
			exprAttrNames[placeholderName] = parser.AttributeName
			marshaledVal, err := attributevalue.Marshal(fieldValue.Interface())
			if err != nil {
				return fmt.Errorf("failed to marshal field %s: %w", field.Name, err)
			}
			exprAttrValues[placeholderValue] = marshaledVal
		}
	}
	if keyAttr == "" {
		return fmt.Errorf("no hash key defined in struct")
	}
	if len(updateExpressions) == 0 {
		return fmt.Errorf("no updatable fields found")
	}
	updateExpr := "SET " + strings.Join(updateExpressions, ", ")

	// Add a condition to ensure that the item exists.
	keyPlaceholder := "#k"
	exprAttrNames[keyPlaceholder] = keyAttr
	conditionExpr := fmt.Sprintf("attribute_exists(%s)", keyPlaceholder)

	input := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       keyMap,
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ConditionExpression:       aws.String(conditionExpr),
	}

	_, err := r.client.UpdateItem(ctx, input)
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to update item: %w", err)
	}
	return nil
}

// Delete deletes an item from DynamoDB by its primary key id (assumed to be of type string).
// This method assumes that the primary key attribute in the table is named "id".
// A conditional expression is used to ensure that the item exists.
func (r *Repository) Delete(ctx context.Context, id string) error {
	// Marshal the id value.
	idAttr, err := attributevalue.Marshal(id)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	// Build the key map – here we assume the primary key attribute is "id".
	key := map[string]types.AttributeValue{
		"id": idAttr,
	}

	input := &dynamodb.DeleteItemInput{
		TableName:           aws.String(r.tableName),
		Key:                 key,
		ConditionExpression: aws.String("attribute_exists(id)"),
	}

	_, err = r.client.DeleteItem(ctx, input)
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete item: %w", err)
	}
	return nil
}
