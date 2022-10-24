module p2

go 1.19

replace ms => ./ms

replace gf => ./gf

replace ra => ./ra

require (
	gf v0.0.0-00010101000000-000000000000
	ra v0.0.0-00010101000000-000000000000
)

require ms v0.0.0-00010101000000-000000000000 // indirect
