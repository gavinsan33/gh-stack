.PHONY: build install

build:
	go build -o gh-stack .

# Replaces any installed gh stack extension with a link to this checkout.
install: build
	-gh extension remove stack
	gh extension install .
