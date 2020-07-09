package main

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	libredis "github.com/go-redis/redis/v7"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type NoSQL struct {
	client *mongo.Client
	test   *mongo.Collection
}

func (noSQL *NoSQL) init() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:example@localhost:27017"))
	check(err)
	collection := client.Database("test").Collection("test")
	noSQL.client = client
	noSQL.test = collection
}

func (noSQL *NoSQL) insert(data interface{}) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := noSQL.test.InsertOne(ctx, data)
	check(err)
}

func (noSQL *NoSQL) count(service string, tag string) int {
	findOptions := options.Find()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	cur, err := noSQL.test.Find(ctx, bson.D{{"service", service}, {"tag", tag}}, findOptions)
	check(err)
	defer cur.Close(ctx)
	count := 0
	for cur.Next(ctx) {
		count++
	}
	return count
}

type Log struct {
	Service string
	IP      string
	Tag     string
}

type PostLogsReqBody struct {
	Service string `json:"service"`
	Tag     string `json:"tag"`
}

type GetLogsReqQuery struct {
	Service string `form:"service"`
	Tag     string `form:"tag"`
}

func initRedis() *libredis.Client {
	option, err := libredis.ParseURL("redis://localhost:6379/0")
	check(err)
	client := libredis.NewClient(option)
	return client
}

func initRateLimit(client *libredis.Client) gin.HandlerFunc {
	// Define a limit rate to 4 requests per hour.
	rate, err := limiter.NewRateFromFormatted("1000-H")
	check(err)

	// Create a store with the redis client.
	store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix:   "AD_Service_limiter",
		MaxRetry: 3,
	})
	check(err)

	// Create a new middleware with the limiter instance.
	middleware := mgin.NewMiddleware(limiter.New(store, rate))
	return middleware
}

// Route
func postLogs(noSQLHandler NoSQL) func(c *gin.Context) {
	return func(c *gin.Context) {
		var body PostLogsReqBody
		err := c.BindJSON(&body)
		check(err)
		noSQLHandler.insert(&Log{
			Service: body.Service,
			IP:      c.ClientIP(),
			Tag:     body.Tag,
		})
		c.Status(204)
	}
}

func getLogsCount(noSQLHandler NoSQL) func(c *gin.Context) {
	return func(c *gin.Context) {
		var query GetLogsReqQuery
		err := c.Bind(&query)
		check(err)
		count := noSQLHandler.count(query.Service, query.Tag)
		c.JSON(200, gin.H{
			"count": count,
		})
	}
}

func main() {
	// Init
	r := gin.Default()
	var noSQLHandler NoSQL = NoSQL{}
	noSQLHandler.init()
	reditClient := initRedis()
	rateLimitMiddleware := initRateLimit(reditClient)
	r.ForwardedByClientIP = true
	r.Use(rateLimitMiddleware)

	// Route
	r.POST("/logs", postLogs(noSQLHandler))
	r.GET("/logs/count", getLogsCount(noSQLHandler))
	r.Run()
}
