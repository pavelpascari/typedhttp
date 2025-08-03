module migration-from-gin

go 1.21

replace github.com/pavelpascari/typedhttp => ../../../

require github.com/pavelpascari/typedhttp v0.0.0-00010101000000-000000000000

// For Gin comparison (build ignored)
require github.com/gin-gonic/gin v1.9.1