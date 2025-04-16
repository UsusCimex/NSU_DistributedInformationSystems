module worker

go 1.24

require (
	common v0.0.0
	github.com/streadway/amqp v1.1.0
)

require go.mongodb.org/mongo-driver v1.17.3 // indirect

replace common => ../common
