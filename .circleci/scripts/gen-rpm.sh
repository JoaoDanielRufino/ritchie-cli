#!/bin/sh

changelog init

mkdir -p pkg-build/SPECS

echo ${RELEASE_VERSION}

mkdir -p dist/installer

go-bin-rpm generate-spec --file packaging/rpm/rpm-team.json -a amd64 --version ${RELEASE_VERSION} > pkg-build/SPECS/ritchiecli.spec
go-bin-rpm generate --file packaging/rpm/rpm-team.json -a amd64 --version ${RELEASE_VERSION} -o dist/installer/ritchie-team.rpm

rm -rf pkg-build && mkdir -p pkg-build/SPECS

go-bin-rpm generate-spec --file packaging/rpm/rpm-single.json -a amd64 --version ${RELEASE_VERSION} > pkg-build/SPECS/ritchiecli.spec
go-bin-rpm generate --file packaging/rpm/rpm-single.json -a amd64 --version ${RELEASE_VERSION} -o dist/installer/ritchie-single.rpm

rm -rf pkg-build && mkdir -p pkg-build/SPECS

go-bin-rpm generate-spec --file packaging/rpm/rpm-team-zup.json -a amd64 --version ${RELEASE_VERSION} > pkg-build/SPECS/ritchiecli.spec
go-bin-rpm generate --file packaging/rpm/rpm-team-zup.json -a amd64 --version ${RELEASE_VERSION} -o dist/installer/ritchie-team-zup.rpm

