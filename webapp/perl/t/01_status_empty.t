use Test::More;
use Test::Deep;
use Isu7final::Game;
use Isu7final::Exponential;

my $got = Isu7final::Game::calc_status(0, {}, [], []);

cmp_deeply $got, superhashof({
    time     => 0,
    items    => [],
    adding   => [],
    schedule => [{
        time        => 0,
        milli_isu   => Isu7final::Exponential->new(0, 0),
        total_power => Isu7final::Exponential->new(0, 0),
    }],
    on_sale  => [],
});

done_testing;
