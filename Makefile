all: clean build

build:
	cd src ; \
	go build -o url ; \
	zip ../Url-Alfred.alfredworkflow . -r --exclude=*.DS_Store* --exclude=.git/* --exclude=*.go --exclude=go.* --exclude="LICENSE" --exclude=".*"

clean:
	rm -f *.alfredworkflow