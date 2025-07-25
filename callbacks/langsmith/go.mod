module github.com/cloudwego/eino-ext/callbacks/langsmith

go 1.23.0

require (
	github.com/bytedance/mockey v1.2.13
	github.com/bytedance/sonic v1.13.2
	github.com/cloudwego/eino v0.3.27
	github.com/cloudwego/eino-ext/libs/acl/langsmith v0.0.0-00010101000000-000000000000
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.10.0
)

// 这是一个本地依赖，需要替换为您本地的路径
replace github.com/cloudwego/eino-ext/libs/acl/langsmith => ../../libs/acl/langsmith
