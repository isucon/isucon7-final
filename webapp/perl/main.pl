#!/usr/bin/env perl

use FindBin;
use lib "$FindBin::Bin/lib";
use Isu7final::Web;

Isu7final::Web->app->start;
