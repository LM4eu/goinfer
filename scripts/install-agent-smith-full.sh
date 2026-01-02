#!/bin/bash

echo "
to remove all globally installed npm packages:

    npm list -g -p | cut -d/ -f7-8 | xargs npm uninstall -g
"

set -xe

# --prefer-offline = use cache first, else download for internet
# --prefer-online = download for internet, except 
npm install --global --prefer-online --no-audit --no-fund \
    @agent-smith/body@latest \
    @agent-smith/brain@latest \
    @agent-smith/cli@latest \
    @agent-smith/feat-git@latest \
    @agent-smith/feat-inference@latest \
    @agent-smith/feat-lang@latest \
    @agent-smith/feat-vision@latest \
    @agent-smith/jobs@latest \
    @agent-smith/lmtask@latest \
    @agent-smith/smem@latest \
    @agent-smith/tfm@latest \
    @agent-smith/tmem-jobs@latest \
    @agent-smith/tmem@latest \
    @locallm/api@latest \
    @locallm/browser@latest \
    @locallm/types@latest \
    termollama@latest \
    \
    @intrinsicai/gbnfgen@latest \
    ;

# to install npm-check-updates@latest
# prefer your distro packet manager:
# sudo pacman -S npm-check-updates

# outdated:
#    @agent-smith/feat-search@latest
#
# removed:
#   @agent-smith/feat-models@latest


# @intrinsicai/gbnfgen = compiler des grammaires gbnf à partir de définitions sour forme d'interfaces typescript
# npm-check-updates = ncu
