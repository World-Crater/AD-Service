# AD-Service

## Requirements

* golang
* air: [How install](https://github.com/cosmtrek/air)

## How to run

1. Set `.env` file
2. Run command line
    ```bash
    $ docker-compose up
    $ go mod init
    $ air
    ```

## Use `adminmongo`

1. Go to web page: `localhost:1234`
2. Typing `Connection name` & `Connection string` from .env file. (e.g:`mongodb://root:example@mongo:27017`)