package dynamodb_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	db "github.com/yuki5155/go-aws/dynamodb"
)

func loadEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

type User struct {
	ID        string `json:"id" dynamodbav:"id" dynamo:"id,key=hash"`
	Email     string `json:"email" dynamodbav:"email" dynamo:"email,required,index=email-index"`
	Name      string `json:"name" dynamodbav:"name" dynamo:"name,required"`
	CreatedAt int64  `json:"created_at" dynamodbav:"created_at" dynamo:"created_at"`
}

func (u *User) TableName() string {
	return "Users"
}
func setupDynamoDBClient(t *testing.T) *dynamodb.Client {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: os.Getenv("AWS_ENDPOINT_URL"),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-northeast-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"localstack",
			"localstack",
			"",
		)),
	)
	if err != nil {
		t.Fatalf("unable to load SDK config: %v", err)
	}

	return dynamodb.NewFromConfig(cfg)
}
func generateTestUser(prefix string) *User {
	return &User{
		ID:        fmt.Sprintf("%s-%s", prefix, time.Now().Format("20060102150405.000")),
		Email:     fmt.Sprintf("test-%s@example.com", time.Now().Format("20060102150405.000")),
		Name:      fmt.Sprintf("Test User %s", time.Now().Format("20060102150405.000")),
		CreatedAt: time.Now().Unix(),
	}
}

func TestRepository_Create_Integration(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	client := setupDynamoDBClient(t)
	repo := db.NewRepository(client, "Users")

	t.Run("Debug direct DynamoDB access", func(t *testing.T) {
		testUser := generateTestUser("direct")

		// デバッグ: 直接DynamoDBへの書き込みを試行
		av, err := attributevalue.MarshalMap(testUser)
		require.NoError(t, err)
		t.Logf("Marshaled item: %+v\n", av)

		input := &dynamodb.PutItemInput{
			TableName: aws.String("Users"),
			Item:      av,
		}
		t.Logf("PutItem input: %+v\n", input)

		_, err = client.PutItem(context.Background(), input)
		require.NoError(t, err, "Direct PutItem failed")
		t.Log("Direct PutItem succeeded")

		// クリーンアップ
		defer func() {
			_, err = client.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
				TableName: aws.String("Users"),
				Key: map[string]types.AttributeValue{
					"id": &types.AttributeValueMemberS{Value: testUser.ID},
				},
			})
			assert.NoError(t, err)
		}()
	})

	t.Run("Create and verify user", func(t *testing.T) {
		testUser := generateTestUser("repo")

		// Create using repository
		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err, "Repository Create failed")

		// Verify
		result, err := client.GetItem(context.Background(), &dynamodb.GetItemInput{
			TableName: aws.String("Users"),
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{Value: testUser.ID},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, result.Item, "Retrieved item should not be nil")
		t.Logf("Retrieved item: %+v\n", result.Item)

		// Verify ID
		if id, ok := result.Item["id"].(*types.AttributeValueMemberS); ok {
			assert.Equal(t, testUser.ID, id.Value)
		} else {
			t.Error("ID field not found or not a string")
		}

		// 重複チェック
		err = repo.Create(context.Background(), testUser)
		assert.ErrorIs(t, err, db.ErrDuplicateKey)

		// クリーンアップ
		// defer func() {
		// 	_, err = client.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		// 		TableName: aws.String("Users"),
		// 		Key: map[string]types.AttributeValue{
		// 			"id": &types.AttributeValueMemberS{Value: testUser.ID},
		// 		},
		// 	})
		// 	assert.NoError(t, err)
		// }()
	})

	t.Run("Validate required fields", func(t *testing.T) {
		invalidUser := &User{
			ID:        generateTestUser("invalid").ID,
			Email:     "", // required フィールドを空に
			Name:      "", // required フィールドを空に
			CreatedAt: time.Now().Unix(),
		}

		err := repo.Create(context.Background(), invalidUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})
	// create valid user
	t.Run("Create valid user", func(t *testing.T) {
		validUser := &User{
			ID:        generateTestUser("valid").ID,
			Email:     "example@example.com",
			Name:      "example",
			CreatedAt: time.Now().Unix(),
		}
		err := repo.Create(context.Background(), validUser)
		assert.NoError(t, err)
	})

}
