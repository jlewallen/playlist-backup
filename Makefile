all: playlist-backup

playlist-backup: *.go
	go build -o $@ $^

clean:
	rm -f playlist-backup
