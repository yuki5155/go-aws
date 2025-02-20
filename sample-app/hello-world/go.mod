require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/yuki5155/go-aws v0.0.0-unpublished
)

replace (
	github.com/yuki5155/go-aws v0.0.0-unpublished => ../../
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
)

module hello-world

go 1.23.4
