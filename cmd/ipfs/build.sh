function build_mipsle(){
    GOOS=linux GOARCH=mipsle go build -o bin/ipfs_mipsle .
    echo "mipsle successed."
}
function build_mips(){
    GOOS=linux GOARCH=mips go build -o bin/ipfs_mips .
    echo "mips successed."
}
function build_arm(){
    GOOS=linux GOARCH=arm GOARM=5 go build -o bin/ipfs_arm .
    echo "arm successed."
}
function build_linux(){
    GOOS=linux GOARCH=amd64 go build -o bin/ipfs_linux .
    echo "amd64 successed."
}
function build_darwin(){
    go build -o bin/ipfs_darwin .
    echo "darwin successed."
}


if [[ -n $1 ]] && [[ $1 = 'all' ]]; then
    echo "build all"
    rm -rf bin
    mkdir bin
    build_mipsle
    build_mips
    build_arm
    build_linux
    build_darwin
    chmod 755 bin/*
elif [[ -n $1 ]] && [[ $1 = 'mipsle' ]]; then
    echo "build mipsle"
    build_mipsle
elif [[ -n $1 ]] && [[ $1 = 'mips' ]]; then
    echo "build mips"
    build_mips
elif [[ -n $1 ]] && [[ $1 = 'arm' ]]; then
    echo "build arm"
    build_arm
elif [[ -n $1 ]] && [[ $1 = 'local' ]]; then
    echo "build local"
    go install -v .
    echo "successed."
else 
    echo "-------------------------------------------------"
    echo "build.sh all : arm \ mips \ mipsle ... "
    echo "build.sh local : install to local gopath/bin "
    echo "-------------------------------------------------"
fi

