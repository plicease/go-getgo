#!/usr/bin/env perl

use strict;
use warnings;
use experimental qw( postderef say );

my %target = (
  linux  => [qw( arm64 amd64 )],
  darwin => [qw( arm64 amd64 )],
);

my $fail = 0;

foreach my $os (sort keys %target) {
  foreach my $arch (sort $target{$os}->@*) {
    local $ENV{GOOS} = $os;
    local $ENV{GOARCH} = $arch;
    my @command = ('go', 'build', -o => "bin/getgo-$os-$arch", './cmd/getgo' );
    print "+env GOOS=$os GOARCH=$arch @command\n";
    system @command and $fail =1;
  }
}

if($fail)
{
  die "one or more builds failed";
}
