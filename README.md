## Setup

Initialize the DynamoDB ORM structure:

```go
import(
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    db "github.com/yuki5155/go-aws/dynamodb"
)

cfg, err := config.LoadDefaultConfig(
    context.TODO(),
    config.WithRegion("your_region"),
)

if err != nil {
    t.Fatalf("unable to load SDK config: %v", err)
}
client := dynamodb.NewFromConfig(cfg)
repo := db.NewRepository(client, "table_name")
```

## Create

To create a record in DynamoDB, first define your data structure using the DynamoDB ORM:

```go
type User struct {
    ID        string `json:"id" dynamodbav:"id" dynamo:"id,key=hash"`
    Email     string `json:"email" dynamodbav:"email" dynamo:"email,required,index=email-index"`
    Name      string `json:"name" dynamodbav:"name" dynamo:"name,required"`
    CreatedAt int64  `json:"created_at" dynamodbav:"created_at" dynamo:"created_at"`
}

func (u *User) TableName() string {
    return "Users"
}
```

Then, create a record in DynamoDB:

```go
err := repo.Create(context.Background(), your_user_struct)
if err != nil {
    t.Fatalf("unable to create record: %v", err)
}
```

## Get

To retrieve records from DynamoDB, use the same data structure defined above. You can use the `config` and `repo` instances created in the Setup section.

### Find by ID

```go
var users user
err := repo.FindByID(context.Background(), "your_id", &users)
```

### Global Secondary Index

```go
var users []user
err := repo.FindByParameter(context.Background(), "your_parameter", "your_value", &users)
```
