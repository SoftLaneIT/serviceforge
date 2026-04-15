module github.com/SoftLaneIT/serviceforge/services/tenant-service

go 1.23.0

require (
	github.com/SoftLaneIT/serviceforge/packages/go-common v0.0.0
	github.com/jackc/pgx/v5 v5.6.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/SoftLaneIT/serviceforge/packages/go-common => ../../packages/go-common
