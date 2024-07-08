# Gyr

A package intended for my own personal use. Currently contains a request router and a database migrator.

## Installation

```sh
go get -u github.com/aigr20/gyr
```

## Debugging

In order to get debug logs from Gyr the environment variable GYR_DEBUG must be set when running your program.

```sh
GYR_DEBUG= go run main.go
```

## Examples

### Router

Creating a router with a couple routes with various HTTP methods.

```go
func main() {
    router := gyr.DefaultRouter()
    router.Path("/").Get(RootHandler)
    router.Path("/file").Get(GetFileHandler).Post(CreateFileHandler)
}
```

### Migrator

Initialize migrator with a custom directory to search for SQL scripts in. Default directory is ./migrations.
SQL files should have their names prefixed with a version, separated from the rest of the file name by an underscore. For example: 0.0.1_initial_tables.sql.

```go
func main() {
    migrator := gyr.NewMigrator(dbConnection, gyr.MigrationDirectory("my-sql-scripts"))
    err := migrator.Migrate()
}
```

