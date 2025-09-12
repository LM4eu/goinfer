#!/bin/sh

# This script has been heavily inspired from
# https://github.com/rourken/ubuntu-gpu-ml-setup/blob/main/INSTALL.md

# This script installs the required Nvidia CUDA libraries and tool required by
# for llama-server = https://github.com/ggml-org/llama.cpp/tree/master/tools/server
# (Goinfer requires llama-server)

# The goal is a good balance between:
# - stable environment in production
# - recent Nvidia CUDA libs & tool that officially support Ubuntu-24.04

# If you are already root, unset this sudo variable
sudo=sudo

$sudo apt-get update
$sudo apt-get dist-upgrade --yes

$sudo apt-get install ubuntu-server --yes

apt-get update
apt-get install  -y

$sudo apt-get install --yes \
   autoconf \
   automake \
   build-essential \
   cmake \
   curl \
   gcc \
   gfortran \
   git \
   gnupg \
   iproute2 \
   libaom-dev \
   libass-dev \
   libatlas-base-dev \
   libavcodec-dev \
   libavformat-dev \
   libcurl4-openssl-dev \
   libdav1d-dev \
   libfaac-dev \
   libfdk-aac-dev \
   libfreetype6-dev \
   libgnutls28-dev \
   libgtk-3-dev \
   libgtk2.0-dev \
   libjpeg-dev \
   libjpeg8-dev \
   libmp3lame-dev \
   libnuma-dev \
   libomp-dev \
   libopencore-amrnb-dev \
   libopencore-amrwb-dev \
   libopus-dev \
   libpng-dev \
   libprotobuf-dev \
   libsdl2-dev \
   libswscale-dev \
   libtbb-dev \
   libtheora-dev \
   libtiff-dev \
   libtool \
   libunistring-dev \
   libv4l-dev \
   libva-dev \
   libvdpau-dev \
   libvorbis-dev \
   libvpx-dev \
   libx264-dev \
   libx265-dev \
   libxcb-shm0-dev \
   libxcb-xfixes0-dev \
   libxcb1-dev \
   libxvidcore-dev \
   nasm \
   neofetch \
   net-tools \
   pciutils \
   pkg-config \
   portaudio19-dev \
   protobuf-compiler \
   pyqt5-dev-tools \
   python3-dev \
   python3-packaging \
   python3-pip \
   python3-venv \
   qt5-qmake \
   qtbase5-dev \
   qtbase5-dev-tools \
   qtchooser \
   screen \
   ssh \
   texinfo \
   unzip \
   v4l-utils \
   wget \
   x264 \
   yasm \
   zlib1g-dev \
   ;

$sudo apt-get install --yes \
   btop \
   htop \
   ;

## Remove remaining Nvidia packages and configuration files
dpkg -l | grep -q nvidia &&
$sudo apt-get purge --purge --autoremove cuda-keyring 'nvidia-*' 'libnvidia-*' 'cuda*' 'cudnn*' 'libcudnn*' --yes &&
$sudo rm -rf /usr/local/cuda* /usr/local/nvidia* /etc/apt/sources.list.d/cuda* /etc/apt/sources.list.d/nvidia*

## Add official Nvidia APT repository for Ubuntu 24.04
wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb
$sudo dpkg -i cuda-keyring_1.1-1_all.deb
rm cuda-keyring_1.1-1_all.deb
$sudo apt-get update

# ## Install Nvidia drivers and CUDA-12.6
# $sudo apt-get install cuda-drivers cudnn9-cuda-12 cuda-toolkit-12-6 --yes
# 
# ## Install Nvidia drivers and CUDA-12.8
# $sudo apt-get install cuda-drivers cudnn9-cuda-12 cuda-toolkit-12-8 --yes
# 
# ## Install Nvidia drivers and CUDA-12.9
# $sudo apt-get install cuda-drivers cudnn9-cuda-12 cuda-toolkit-12 --yes
#
## Install Nvidia drivers and CUDA-13
$sudo apt-get install cuda-drivers cudnn9-cuda-13 cuda-toolkit-13 --yes

echo '
[ -d /usr/local/cuda-12 ] && export CUDA_HOME=/usr/local/cuda-12
[ -d /usr/local/cuda-13 ] && export CUDA_HOME=/usr/local/cuda-13
export CUDA_TOOLKIT_ROOT_DIR=$CUDA_HOME
export PATH=$CUDA_HOME/bin${PATH:+:${PATH}}
export LD_LIBRARY_PATH=$CUDA_HOME/lib64${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}
' >> ~/.profile

echo "
Now the computer needs to reboot, do:
$sudo reboot
"
