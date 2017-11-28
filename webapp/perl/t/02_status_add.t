use Test::More;
use Test::Deep;
use Isu7final::Game;
use Isu7final::Exponential;

my $m_items = {};
my $addings = [
    { time => 100, isu => 1 },
    { time => 200, isu => 2 },
    { time => 300, isu => 1234567890123456789 },
];
my $buyings = [];

{
    my $got = Isu7final::Game::calc_status(0, $m_items, $addings, $buyings);
    
    cmp_deeply $got, superhashof({
        time     => 0,
        items    => [],
        adding   => bag(
            superhashof({ time => 100, isu => 1 }),
            superhashof({ time => 200, isu => 2 }),
            superhashof({ time => 300, isu => 1234567890123456789 }),
        ),
        schedule => [
            {
                time        => 0,
                milli_isu   => Isu7final::Exponential->new(0, 0),
                total_power => Isu7final::Exponential->new(0, 0),
            },
            {
                time        => 100,
                milli_isu   => Isu7final::Exponential->new(1000, 0),
                total_power => Isu7final::Exponential->new(0, 0),
            },
            {
                time        => 200,
                milli_isu   => Isu7final::Exponential->new(3000, 0),
                total_power => Isu7final::Exponential->new(0, 0),
            },
            {
                time        => 300,
                milli_isu   => Isu7final::Exponential->new(123456789012345, 7),
                total_power => Isu7final::Exponential->new(0, 0),
            },
        ],
        on_sale  => [],
    });
}

{
    my $got = Isu7final::Game::calc_status(500, $m_items, $addings, $buyings);

    cmp_deeply $got, superhashof({
        time     => 0,
        items    => [],
        adding   => [],
        schedule => [
            {
                time        => 500,
                milli_isu   => Isu7final::Exponential->new(123456789012345, 7),
                total_power => Isu7final::Exponential->new(0, 0),
            },
        ],
        on_sale  => [],
    });
}

done_testing;
