# Goinfer

Inference api server for local gguf language models. Based on [llama.cpp](https://github.com/ggml-org/llama.cpp) and [llama-swap](https://github.com/mostlygeek/llama-swap).

- **Multi models**: switch between models at runtime
- **Inference queries**: http api and streaming response support

Works with the [Infergui](https://github.com/synw/infergui) frontend

<details>
<summary>:books: Read the <a href="https://synw.github.io/goinfer/">documentation</a></summary>

 - [Get started](https://synw.github.io/goinfer/get_started)
    - [Install](https://synw.github.io/goinfer/get_started/install)
    - [Configure](https://synw.github.io/goinfer/get_started/configure)
    - [Run](https://synw.github.io/goinfer/get_started/run)
 - [Llama api](https://synw.github.io/goinfer/llama_api)
    - [Models state](https://synw.github.io/goinfer/llama_api/models_state)
    - [Load model](https://synw.github.io/goinfer/llama_api/load_model)
    - [Inference](https://synw.github.io/goinfer/llama_api/inference)
    - [Tasks](https://synw.github.io/goinfer/llama_api/tasks)
    - [Templates](https://synw.github.io/goinfer/llama_api/templates)
 - [Openai api](https://synw.github.io/goinfer/openai_api)
    - [Configure](https://synw.github.io/goinfer/openai_api/configure)
    - [Endpoints](https://synw.github.io/goinfer/openai_api/endpoints)

</details>

# System Requirements and Dependencies

Goinfer requires [`llama-server`](https://github.com/ggml-org/llama.cpp/tree/master/tools/server).
`llama-server` infers faster on GPU. Below is the recipe to get a reproducible production environment based on Nvidia CUDA that officially supports Ubuntu-24.04. The following instructions have been inspired from https://github.com/rourken/ubuntu-gpu-ml-setup/blob/main/INSTALL.md

```
sudo apt-get update
sudo apt-get dist-upgrade --yes

sudo apt-get install ubuntu-server --yes

sudo apt-get install --yes \
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
   pkg-config \
   portaudio19-dev \
   protobuf-compiler \
   pyqt5-dev-tools \
   python3 \
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

sudo apt-get install --yes \
   btop \
   htop \
   ;

## Remove remaining Nvidia packages and configuration files

dpkg -l | grep -q nvidia &&
sudo apt-get purge --purge --autoremove cuda-keyring 'nvidia-*' 'libnvidia-*' 'cuda*' 'cudnn*' 'libcudnn*' --yes &&
sudo rm -rf /usr/local/cuda* /usr/local/nvidia* /etc/apt/sources.list.d/cuda* /etc/apt/sources.list.d/nvidia*

wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb
sudo dpkg -i cuda-keyring_1.1-1_all.deb
rm cuda-keyring_1.1-1_all.deb
sudo apt-get update

# ## Install latest Nvidia drivers and CUDA-12.6
# 
# sudo apt-get install cuda-drivers cudnn9-cuda-12 cuda-toolkit-12-6 --yes
# 
# ## Install latest Nvidia drivers and CUDA-12.8
# 
# sudo apt-get install cuda-drivers cudnn9-cuda-12 cuda-toolkit-12-8 --yes
# 
# ## Install latest Nvidia drivers and CUDA-12.9
# 
# sudo apt-get install cuda-drivers cudnn9-cuda-12 cuda-toolkit-12 --yes

## Install latest Nvidia drivers and CUDA-13

sudo apt-get install cuda-drivers cudnn9-cuda-13 cuda-toolkit-13 --yes

echo '
[ -d /usr/local/cuda-12 ] && export CUDA_HOME=/usr/local/cuda-12
[ -d /usr/local/cuda-13 ] && export CUDA_HOME=/usr/local/cuda-13
export CUDA_TOOLKIT_ROOT_DIR=$CUDA_HOME
export PATH=$CUDA_HOME/bin${PATH:+:${PATH}}
export LD_LIBRARY_PATH=$CUDA_HOME/lib64${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}
' >> ~/.profile

sudo reboot
```
