bench:
	go test -run none -bench .

bench-old:
	go test -run none -bench . >old.txt

bench-new:
	go test -run none -bench . >new.txt

benchcomp:
	benchcmp old.txt new.txt
clean:
	rm -f *out *.test

cpu: clean
	go test -run @ -bench . -cpuprofile cpu.out
	go tool pprof -lines *.test cpu.out

mem: clean
	go test -run @ -bench . -memprofile mem.out -memprofilerate 1 -timeout 24h
	go tool pprof -lines  -alloc_space *.test mem.out

