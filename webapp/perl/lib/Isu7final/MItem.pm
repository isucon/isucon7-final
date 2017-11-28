package Isu7final::MItem;

use Mojo::Base -base;
use Math::BigInt lib => "GMP";

has [qw(
    item_id
    price1 price2 price3 price4
    power1 power2 power3 power4
)];

# power(x):=(cx+1)*d^(ax+b)
sub get_power {
    my ($self, $count) = @_;

    my $a = $self->power1;
    my $b = $self->power2;
    my $c = $self->power3;
    my $d = $self->power4;
    my $x = $count;

    my $s = Math::BigInt->new($c*$x + 1);
    my $t = Math::BigInt->new($d)->bpow(Math::BigInt->new($a*$x+$b));

    return $s->bmul($t);
}

# price(x):=(cx+1)*d^(ax+b)
sub get_price {
    my ($self, $count) = @_;

    my $a = $self->price1;
    my $b = $self->price2;
    my $c = $self->price3;
    my $d = $self->price4;
    my $x = $count;

    my $s = Math::BigInt->new($c*$x + 1);
    my $t = Math::BigInt->new($d)->bpow(Math::BigInt->new($a*$x+$b));

    return $s->bmul($t);
}

1;
