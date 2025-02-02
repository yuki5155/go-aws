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

// DynamoTagParser カスタムタグのパーサー構造体
type DynamoTagParser struct {
	AttributeName string
	KeyType       string
	Index         string
	Required      bool
}

// TableNamer テーブル名を提供するインターフェース
type TableNamer interface {
	TableName() string
}

// ParseDynamoTag タグの文字列をパースする関数
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

// DynamoDBClient インターフェース
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// Repository DynamoDB操作の構造体
type Repository struct {
	client    DynamoDBClient
	tableName string // デフォルトテーブル名
}

// カスタムエラーの定義
var (
	ErrDuplicateKey = errors.New("item with this key already exists")
)

// NewRepository リポジトリのコンストラクタ
func NewRepository(client DynamoDBClient, defaultTableName string) *Repository {
	return &Repository{
		client:    client,
		tableName: defaultTableName,
	}
}

// getTableName テーブル名を取得する関数
func (r *Repository) getTableName(item interface{}) string {
	// TableNamerインターフェースが実装されているか確認
	if tableNamer, ok := item.(TableNamer); ok {
		return tableNamer.TableName()
	}

	// ポインタの場合は中身を確認
	if reflect.ValueOf(item).Kind() == reflect.Ptr {
		if tableNamer, ok := reflect.ValueOf(item).Elem().Interface().(TableNamer); ok {
			return tableNamer.TableName()
		}
	}

	// インターフェースが実装されていない場合はデフォルトテーブル名を使用
	return r.tableName
}

// Create レコードを作成する関数
func (r *Repository) Create(ctx context.Context, item interface{}) error {
	// 構造体の検証
	if err := validateStruct(item); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// DynamoDB Item に変換
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	// テーブル名の取得
	tableName := r.getTableName(item)

	// 条件式の作成（既存キーがない場合のみ作成）
	conditionExpression := createConditionExpression(item)

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                av,
		ConditionExpression: aws.String(conditionExpression),
	}

	// PutItem実行
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

// validateStruct 構造体の検証
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

// createConditionExpression 条件式の作成
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
