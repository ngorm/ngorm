PG=postgres://postgres@localhost:5432/ngorm?sslmode=disable

bench:
	go test -run none -bench .

bench-old:
	NGORM_PG_CONN=$(PG) go test -run none -bench . >old.txt

bench-new:
	NGORM_PG_CONN=$(PG) go test -run none -bench . >new.txt

benchcomp:
	benchcmp old.txt new.txt
clean:
	rm -f *out *.test *.txt

cpu: clean
	NGORM_PG_CONN=$(PG) go test -run @ -bench . -cpuprofile cpu.out
	go tool pprof -lines *.test cpu.out

mem: clean
	NGORM_PG_CONN=$(PG) go test -run @ -bench . -memprofile mem.out -memprofilerate 1 -timeout 24h
	go tool pprof -lines  -alloc_objects *.test mem.out

