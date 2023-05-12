require 'sinatra'
require 'sequel'

# Configure a connection pool that connects to the proxy via TCP
def connect_tcp
    Sequel.connect(
        adapter: 'postgres',
        host: ENV["INSTANCE_HOST"],
        database: ENV["DB_NAME"],
        user: ENV["DB_USER"],
        password: ENV["DB_PASS"],
        pool_timeout: 5,
        max_connections: 5,
    )
end

DB = connect_tcp()


get '/' do
    # Connect to the database and get the current time
    DB["SELECT NOW()"].all
end
