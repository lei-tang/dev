# don't put duplicate lines or lines starting with space in the history.
# See bash(1) for more options
HISTCONTROL=ignoreboth

# append to the history file, don't overwrite it
shopt -s histappend
# for setting history length see HISTSIZE and HISTFILESIZE in bash(1)
HISTSIZE=1000
HISTFILESIZE=2000

# check the window size after each command and, if necessary,
# update the values of LINES and COLUMNS.
shopt -s checkwinsize

# If set, the pattern "**" used in a pathname expansion context will
# match all files and zero or more directories and subdirectories.
#shopt -s globstar

# make less more friendly for non-text input files, see lesspipe(1)
[ -x /usr/bin/lesspipe ] && eval "$(SHELL=/bin/sh lesspipe)"

# set variable identifying the chroot you work in (used in the prompt below)
#if [ -z "${debian_chroot:-}" ] && [ -r /etc/debian_chroot ]; then
#    debian_chroot=$(cat /etc/debian_chroot)
#fi

# set a fancy prompt (non-color, unless we know we "want" color)
case "$TERM" in
    xterm-color|*-256color) color_prompt=yes;;
esac

# uncomment for a colored prompt, if the terminal has the capability; turned
# off by default to not distract the user: the focus in a terminal window
# should be on the output of commands, not on the prompt
#force_color_prompt=yes

if [ -n "$force_color_prompt" ]; then
    if [ -x /usr/bin/tput ] && tput setaf 1 >&/dev/null; then
	# We have color support; assume it's compliant with Ecma-48
	# (ISO/IEC-6429). (Lack of such support is extremely rare, and such
	# a case would tend to support setf rather than setaf.)
	color_prompt=yes
    else
	color_prompt=
    fi
fi

if [ "$color_prompt" = yes ]; then
    PS1='${debian_chroot:+($debian_chroot)}\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '
else
    PS1='${debian_chroot:+($debian_chroot)}\u@\h:\w\$ '
fi
unset color_prompt force_color_prompt

# If this is an xterm set the title to user@host:dir
case "$TERM" in
xterm*|rxvt*)
    PS1="\[\e]0;${debian_chroot:+($debian_chroot)}\u@\h: \w\a\]$PS1"
    ;;
*)
    ;;
esac

export PS1="${debian_chroot:+($debian_chroot)}\[\e[1;31m\]\u\[\e[m\]\[\e[01m\][\[\033[1;33m\]\T \d\[\e[m\]\[\e[01m\]]:\[\e[1;36m\]\w\[\e[01m\]\[\e[m\]\n\[\e[0;32m\]~\$ "

# enable color support of ls and also add handy aliases
if [ -x /usr/bin/dircolors ]; then
    test -r ~/.dircolors && eval "$(dircolors -b ~/.dircolors)" || eval "$(dircolors -b)"
    alias ls='ls --color=auto'
    #alias dir='dir --color=auto'
    #alias vdir='vdir --color=auto'

    alias grep='grep --color=auto'
    alias fgrep='fgrep --color=auto'
    alias egrep='egrep --color=auto'
fi

export CLICOLOR=1
#export LSCOLORS=ExFxCxDxBxegedabagacad
#you can use this if you are using a black background:
export LSCOLORS=gxBxhxDxfxhxhxhxhxcxcx

# colored GCC warnings and errors
#export GCC_COLORS='error=01;31:warning=01;35:note=01;36:caret=01;32:locus=01:quote=01'


#autojump config
[ -f /usr/local/etc/profile.d/autojump.sh ] && . /usr/local/etc/profile.d/autojump.sh
# Alias definitions.
# You may want to put all your additions into a separate file like
# ~/.bash_aliases, instead of adding them here directly.
# See /usr/share/doc/bash-doc/examples in the bash-doc package.

if [ -f ~/.bash_aliases ]; then
    . ~/.bash_aliases
fi

alias clion='/Applications/CLion.app/Contents/MacOS/clion'
#beyond compare
alias bcomp='/Applications/Beyond\ Compare.app/Contents/MacOS/bcomp'

# enable programmable completion features (you don't need to enable
# this, if it's already enabled in /etc/bash.bashrc and /etc/profile
# sources /etc/bash.bashrc).
# On the bash_completion installed on osx through brew, the path is:
if [ -f $(brew --prefix)/etc/bash_completion ]; then
. $(brew --prefix)/etc/bash_completion
fi

export ANDROID_HOME=/Users/$USER/Library/Android/sdk
export TENSORFLOW_HOME=/Users/$USER/tensorflow
export PATH=${PATH}:$ANDROID_HOME/tools:$ANDROID_HOME/platform-tools:$TENSORFLOW_HOME/bin:$HOME/.rvm/bin
#set the python path for YouCompleteMe plugin in vim to find the python packages.
#export PYTHONPATH=$TENSORFLOW_HOME/lib/python2.7/site-packages

# add go to path
export GOROOT=/usr/local/go
export PATH=$GOROOT/bin:$PATH

# Every go project must has its own GOPATH setting for golang to find its dependency.
# When switching go projects, point GOPATH to the new go project.
# Note: As desktop launcher will not reload .bashrc, after GOPATH changes, need
# to launch IntelliJ from a terminal instead of double clicking the
# IntelliJ icon on the desktop to pick up the updated path.
#export GOPATH=/usr/lib/google-golang:~/myfiles/learn/cloud/Kubernetes/CRD/example1/git/hello-world
#export GOPATH=~/myfiles/learn/cloud/Kubernetes/CRD/example1/git/hello-world:~/myfiles/learn/cloud/Kubernetes/CRD/example1/git/kube-crd
export GOPATH=~/go
# Every go project has src, bin, and pkg directories
PATH=$PATH:${GOPATH//://bin:}/bin
#export PATH=$GOPATH/bin:$PATH

export ISTIO=$GOPATH/src/istio.io
export MIXER_REPO=$GOPATH/src/istio.io/istio/mixer

# Please change HUB to the desired HUB for custom docker container
# builds.
export HUB="docker.io/$USER"

# The Istio Docker build system will build images with a tag composed of
# $USER and timestamp. The codebase doesn't consistently use the same timestamp
# tag. To simplify development the development process when later using
# updateVersion.sh you may find it helpful to set TAG to something consistent
# such as $USER.
export TAG=$USER

# If your github username is not the same as your local user name (saved in the
# shell variable $USER), then replace "$USER" below with your github username
export GITHUB_USER=<fill your user name>

# Specify which Kube config you'll use for testing. This depends on whether
# you're using Minikube or your own Kubernetes cluster for local testing
# For a GKE cluster:
#export KUBECONFIG=${HOME}/.kube/config
# Alternatively, for Minikube:
# export KUBECONFIG=${GOPATH}/src/istio.io/istio/.circleci/config

#gtest header file and library locations
#/usr/local/include/gtest
#/usr/local/lib/libgtest.a
#Example: g++ Test.o Add.o /usr/local/lib/libgtest.a -lpthread

[ -f ~/.fzf.bash ] && source ~/.fzf.bash

#Add cmake to path
#export CMAKE=/home/tools/cmake/cmake-3.10.2-Linux-x86_64
#export PATH=$PATH:$CMAKE/bin
