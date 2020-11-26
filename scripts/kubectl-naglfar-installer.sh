#!/bin/sh

version=0.2.0

case $(uname -s) in
    Linux|linux) os=linux ;;
    Darwin|darwin) os=darwin ;;
    *) os= ;;
esac

if [ -z "$os" ]; then
    echo "OS $(uname -s) not supported." >&2
    exit 1
fi

binary_url="https://github.com/PingCAP-QE/Naglfar/releases/download/v${version}/Naglfar_${version}_${os}_amd64.tar.gz"

case $(uname -m) in
    amd64|x86_64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) arch= ;;
esac

if [ -z "$NAGLFAR_HOME" ]; then
    NAGLFAR_HOME=$HOME/.Naglfar
fi
bin_dir=$NAGLFAR_HOME/bin
mkdir -p "$bin_dir"

install_binary() {
    curl -L $binary_url -o "/tmp/Naglfar_${version}_${os}_amd64.tar.gz" || return 1
    tar -zxf "/tmp/Naglfar_${version}_${os}_amd64.tar.gz" -C "$bin_dir" || return 1
    rm "/tmp/Naglfar_${version}_${os}_amd64.tar.gz"
    return 0
}

if ! install_binary; then
    echo "Failed to download and/or extract Naglfar archive."
    exit 1
fi

mv "$bin_dir/kubectl-naglfar" "$bin_dir/naglfar"
chmod 755 "$bin_dir/naglfar"


bold=$(tput bold 2>/dev/null)
sgr0=$(tput sgr0 2>/dev/null)

# Refrence: https://stackoverflow.com/questions/14637979/how-to-permanently-set-path-on-linux-unix
shell=$(echo $SHELL | awk 'BEGIN {FS="/";} { print $NF }')
echo "Detected shell: ${bold}$shell${sgr0}"
if [ -f "${HOME}/.${shell}_profile" ]; then
    PROFILE=${HOME}/.${shell}_profile
elif [ -f "${HOME}/.${shell}_login" ]; then
    PROFILE=${HOME}/.${shell}_login
elif [ -f "${HOME}/.${shell}rc" ]; then
    PROFILE=${HOME}/.${shell}rc
else
    PROFILE=${HOME}/.profile
fi
echo "Shell profile:  ${bold}$PROFILE${sgr0}"

case :$PATH: in
    *:$bin_dir:*) : "PATH already contains $bin_dir" ;;
    *) printf 'export PATH=%s:$PATH\n' "$bin_dir" >> "$PROFILE"
        echo "$PROFILE has been modified to to add Naglfar to PATH"
        echo "open a new terminal or ${bold}source ${PROFILE}${sgr0} to use it"
        ;;
esac

echo "Installed path: ${bold}$bin_dir/naglfar${sgr0}"
echo "==============================================="
echo "Have a try:     ${bold}naglfar ${sgr0}"
echo "==============================================="
