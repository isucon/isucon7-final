package Isu7final::Exponential;

# 10進数の指数表記に使うデータ。JSONでは [仮数部, 指数部] という2要素配列になる。

use Mojo::Base -base;
use JSON::Types;

has [qw(mantissa exponent)];

sub new {
    return shift->SUPER::new(mantissa => $_[0], exponent => $_[1]); # Mantissa * 10 ^ Exponent
}

sub TO_JSON {
    my $self = shift;

    return [ number $self->mantissa, number $self->exponent ];
}

1;
