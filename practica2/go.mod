module p2

go 1.19

replace ms => ./ms

replace gf => ./gf

replace ra => ./ra

require (
	gf v0.0.0-00010101000000-000000000000
	ra v0.0.0-00010101000000-000000000000
)

require (
	github.com/bramvdbogaerde/go-scp v1.2.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	ms v0.0.0-00010101000000-000000000000 // indirect
)
