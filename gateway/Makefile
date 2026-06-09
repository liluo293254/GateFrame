.PHONY: build test run clippy build-ddev

build:
	cargo build --release

build-ddev:
	bash build-ddev.sh

test:
	cargo test

clippy:
	cargo clippy -- -D warnings

run:
	cargo run
