package dynamodb_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	db "github.com/yuki5155/go-aws/dynamodb"
)

func createUsersTable(client *dynamodb.Client) error {
	_, err := client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("email"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("email-index"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("email"),
						KeyType:       types.KeyTypeHash,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		TableName: aws.String("Users"),
	})
	return err
}

func TestRepository_FindByID_Integration(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	client := setupDynamoDBClient(t)
	repo := db.NewRepository(client, "Users")
	err := createUsersTable(client)
	if err != nil && !strings.Contains(err.Error(), "Table already exists") {
		t.Fatal(err)
	}
	t.Run("Find existing user", func(t *testing.T) {
		// テストユーザーの作成
		testUser := generateTestUser("find")
		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)

		// FindByID実行
		var foundUser User
		err = repo.FindByID(context.Background(), testUser.ID, &foundUser)
		assert.NoError(t, err)
		assert.Equal(t, testUser.ID, foundUser.ID)
		assert.Equal(t, testUser.Email, foundUser.Email)
		assert.Equal(t, testUser.Name, foundUser.Name)
	})

	t.Run("Find non-existing user", func(t *testing.T) {
		var foundUser User
		err := repo.FindByID(context.Background(), "non-existing-id", &foundUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})
}

func TestRepository_FindByParameter_Integration(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	client := setupDynamoDBClient(t)
	repo := db.NewRepository(client, "Users")

	t.Run("Find by email using index", func(t *testing.T) {
		// テストユーザーの作成
		testUser := generateTestUser("param")
		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)

		// FindByParameter実行
		var users []User
		err = repo.FindByParameter(context.Background(), "email", testUser.Email, &users)
		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, testUser.ID, users[0].ID)
	})

	t.Run("Find by name using scan", func(t *testing.T) {
		testUser := generateTestUser("scan")
		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)

		var users []User
		err = repo.FindByParameter(context.Background(), "name", testUser.Name, &users)
		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, testUser.ID, users[0].ID)
	})

	t.Run("Find with no results", func(t *testing.T) {
		var users []User
		err := repo.FindByParameter(context.Background(), "email", "nonexistent@example.com", &users)
		assert.NoError(t, err)
		assert.Len(t, users, 0)
	})
}

func TestRepository_GetAll_Integration(t *testing.T) {
	if err := loadEnv("../.env"); err != nil {
		t.Fatal(err)
	}

	client := setupDynamoDBClient(t)
	repo := db.NewRepository(client, "Users")

	t.Run("Get all users", func(t *testing.T) {
		// 複数のテストユーザーを作成
		testUsers := []*User{
			generateTestUser("getall1"),
			generateTestUser("getall2"),
			generateTestUser("getall3"),
		}

		for _, user := range testUsers {
			err := repo.Create(context.Background(), user)
			require.NoError(t, err)
		}

		// GetAll実行
		var users []User
		err := repo.GetAll(context.Background(), &users)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), len(testUsers))

		// 作成したユーザーが含まれているか確認
		foundIDs := make(map[string]bool)
		for _, user := range users {
			foundIDs[user.ID] = true
		}
		for _, testUser := range testUsers {
			assert.True(t, foundIDs[testUser.ID])
		}
	})
}
