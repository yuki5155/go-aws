package dynamodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	db "github.com/yuki5155/go-aws/dynamodb"
)

func setupAwsDynamoDBClient(t *testing.T) *dynamodb.Client {

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-northeast-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL: "https://dynamodb.ap-northeast-1.amazonaws.com",
				}, nil
			},
		)),
	)
	if err != nil {
		t.Fatalf("unable to load SDK config: %v", err)
	}

	return dynamodb.NewFromConfig(cfg)
}

type UserDev struct {
	ID        string `json:"id" dynamodbav:"id" dynamo:"id,key=hash"`
	Email     string `json:"email" dynamodbav:"email" dynamo:"email,required,index=email-index"`
	Name      string `json:"name" dynamodbav:"name" dynamo:"name,required,index=name-index"`
	CreatedAt int64  `json:"created_at" dynamodbav:"created_at" dynamo:"created_at"`
}

func (u *UserDev) TableName() string {
	return "Users-dev"
}

func generateAwsTestUser(prefix string) *UserDev {
	return &UserDev{
		ID:        fmt.Sprintf("%s-%s", prefix, time.Now().Format("20060102150405.000")),
		Email:     fmt.Sprintf("test-%s@example.com", time.Now().Format("20060102150405.000")),
		Name:      fmt.Sprintf("Test User %s", time.Now().Format("20060102150405.000")),
		CreatedAt: time.Now().Unix(),
	}
}

func TestAWS_AddUserTable(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	cfg, _ := config.LoadDefaultConfig(context.TODO())
	creds, _ := cfg.Credentials.Retrieve(context.TODO())
	fmt.Println("Credentials:", creds.AccessKeyID)

	client := setupAwsDynamoDBClient(t)
	repo := db.NewRepository(client, "Users-dev")

	t.Run("Create valid user", func(t *testing.T) {
		user := generateAwsTestUser("valid")
		err := repo.Create(context.Background(), user)
		fmt.Println("Create Error:", err)
		assert.NoError(t, err)
	})
	t.Run("Create valid user", func(t *testing.T) {
		user := generateAwsTestUser("valid")
		err := repo.Create(context.Background(), user)
		fmt.Println("Create Error:", err)
		assert.NoError(t, err)

		var found UserDev
		err = repo.FindByID(context.Background(), user.ID, &found)
		fmt.Println("FindByID Error:", err)
		fmt.Println("Found Item:", found)
	})
	t.Run("Create valid user", func(t *testing.T) {
		var users []UserDev
		err := repo.GetAll(context.Background(), &users)
		fmt.Println("Before Create - Items:", len(users))

		user := generateAwsTestUser("valid")
		err = repo.Create(context.Background(), user)
		assert.NoError(t, err)

		err = repo.GetAll(context.Background(), &users)
		fmt.Println("After Create - Items:", len(users))
		fmt.Println("All items:", users)
	})
}

func listTables(client *dynamodb.Client) ([]string, error) {
	var tables []string
	var lastEvaluatedTableName *string

	for {
		input := &dynamodb.ListTablesInput{
			ExclusiveStartTableName: lastEvaluatedTableName,
			Limit:                   aws.Int32(100),
		}

		result, err := client.ListTables(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list tables: %v", err)
		}

		tables = append(tables, result.TableNames...)

		lastEvaluatedTableName = result.LastEvaluatedTableName
		if lastEvaluatedTableName == nil {
			break
		}
	}

	return tables, nil
}

func TestListTables(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}
	client := setupAwsDynamoDBClient(t)

	tables, err := listTables(client)
	for _, tableName := range tables {
		fmt.Println(tableName)
	}
	if err != nil {
		t.Fatalf("failed to list tables: %v", err)
	}

	for _, tableName := range tables {
		t.Logf("Found table: %s", tableName)
	}
}

// get value with id
func TestAWS_GetUser(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	client := setupAwsDynamoDBClient(t)
	repo := db.NewRepository(client, "Users-dev")

	var user UserDev
	err := repo.FindByID(context.Background(), "valid-20250205134058.186", &user)
	assert.NoError(t, err)
	fmt.Println("User:", user)
}

// GlobalSecondaryIndexes
func TestAWS_GetbyEmail(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	client := setupAwsDynamoDBClient(t)
	repo := db.NewRepository(client, "Users-dev")

	// スライスとして定義
	var users []UserDev
	err := repo.FindByParameter(context.Background(), "email", "test-20250205134058.186@example.com", &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1) // スライスの長さをチェック

	if len(users) > 0 {
		fmt.Println("User:", users[0])
	}
}

func TestAws_UpdateUser(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	client := setupAwsDynamoDBClient(t)
	repo := db.NewRepository(client, "Users-dev")

	var user UserDev
	err := repo.FindByID(context.Background(), "valid-20250205134058.186", &user)
	assert.NoError(t, err)
	fmt.Println("User:", user)

	user.Name = "Updated User"
	err = repo.Update(context.Background(), user)
	assert.NoError(t, err)

	var updatedUser UserDev
	err = repo.FindByID(context.Background(), "valid-20250205134058.186", &updatedUser)
	assert.NoError(t, err)
	fmt.Println("Updated User:", updatedUser)
}
