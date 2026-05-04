#!/usr/bin/env bash

# inspired from https://github.com/rectorphp/rector/blob/main/build/build-rector-scoped.sh

# see https://stackoverflow.com/questions/66644233/how-to-propagate-colors-from-bash-script-to-github-action?noredirect=1#comment117811853_66644233
export TERM=xterm-color

# show errors
set -e
set -u


# functions
note()
{
    MESSAGE=$1;
    printf "\n";
    echo "\033[0;33m[NOTE] $MESSAGE\033[0m";
}


# configure here
BUILD_DIRECTORY=$1
RESULT_DIRECTORY=$2


# ---------------------------

# 2. scope it -
note "Downloading php-scoper 0.18.17"
wget https://github.com/humbug/php-scoper/releases/download/0.18.17/php-scoper.phar -N --no-verbose

note "Running php-scoper on /bin, /config/, /src, /vendor and composer.json"
php -d memory_limit=-1 php-scoper.phar add-prefix bin config src vendor composer.json --output-dir "../$RESULT_DIRECTORY" --config scoper.php --force --ansi --working-dir "$BUILD_DIRECTORY";

note "Dumping Composer Autoload"
composer dump-autoload --working-dir "$RESULT_DIRECTORY" --ansi --classmap-authoritative --no-dev

rm -rf "$BUILD_DIRECTORY"

# copy metafiles needed for release
note "Copy metafiles like composer.json, .github etc to repository"
rm -f "$RESULT_DIRECTORY/composer.json"

# make bin files runnable without "php"
chmod 777 "$RESULT_DIRECTORY/bin/ecs"
chmod 777 "$RESULT_DIRECTORY/bin/ecs.php"

note "Finished"
