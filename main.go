package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// User represents a user in our system
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var dynamoClient *dynamodb.Client
var s3Client *s3.Client

func init() {
	// AWS SDK v2の設定
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
		log.Fatal(err)
	}

	// DynamoDBクライアントの初期化
	dynamoClient = dynamodb.NewFromConfig(cfg)
	// S3クライアントの初期化
	s3Client = s3.NewFromConfig(cfg)
}

func main() {
	r := gin.Default()

	// ヘルスチェックエンドポイント
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// ユーザー作成エンドポイント
	r.POST("/users", createUser)

	// ユーザー取得エンドポイント
	r.GET("/users/:id", getUser)

	// S3バケット一覧取得エンドポイント
	r.GET("/buckets", listBuckets)

	log.Fatal(r.Run(":8080"))
}

func createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("Users"),
		Item: map[string]types.AttributeValue{
			"id":   &types.AttributeValueMemberS{Value: user.ID},
			"name": &types.AttributeValueMemberS{Value: user.Name},
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func getUser(c *gin.Context) {
	id := c.Param("id")

	result, err := dynamoClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("Users"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user := User{
		ID:   result.Item["id"].(*types.AttributeValueMemberS).Value,
		Name: result.Item["name"].(*types.AttributeValueMemberS).Value,
	}

	c.JSON(http.StatusOK, user)
}

func listBuckets(c *gin.Context) {
	result, err := s3Client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	buckets := make([]string, 0)
	for _, bucket := range result.Buckets {
		buckets = append(buckets, *bucket.Name)
	}

	c.JSON(http.StatusOK, gin.H{"buckets": buckets})
}
