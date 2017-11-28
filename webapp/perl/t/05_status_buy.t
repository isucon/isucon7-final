use Test::More;
use Test::Deep;
use Isu7final::Game;
use Isu7final::MItem;

my $x = Isu7final::MItem->new(
    item_id => 1,
    power1  => 1, power2  => 1, power3  => 3, power4  => 2,
    price1  => 1, price2  => 1, price3  => 7, price4  => 6,
);

my $y = Isu7final::MItem->new(
    item_id => 2,
    power1  => 1, power2  => 1, power3  => 7, power4  => 6,
    price1  => 1, price2  => 1, price3  => 3, price4  => 2,
);

my $m_items     = { 1 => $x, 2 => $y };
my $initial_isu = 10000000;
my $addings     = [{ time => 0, isu => $initial_isu }];
my $buyings     = [
    { item_id => 1, ordinal => 1, time => 100 },
    { item_id => 1, ordinal => 2, time => 200 },
    { item_id => 2, ordinal => 1, time => 300 },
    { item_id => 2, ordinal => 2, time => 2001 },
];

my $got = Isu7final::Game::calc_status(0, $m_items, $addings, $buyings);

cmp_deeply $got, superhashof({
    time     => 0,
    items    => bag(
        {
            item_id      => 1,
            count_bought => 2,
            count_built  => 0,
            next_price   => Isu7final::Exponential->new(28512, 0),
            power        => Isu7final::Exponential->new(0, 0),
            building     => [
                {
                    time        => 100,
                    count_built => 1,
                    power       => Isu7final::Exponential->new(16, 0),
                },
                {
                    time        => 200,
                    count_built => 2,
                    power       => Isu7final::Exponential->new(72, 0),
                },
            ],
        },
        {
            item_id      => 2,
            count_bought => 2,
            count_built  => 0,
            next_price   => Isu7final::Exponential->new(160, 0),
            power        => Isu7final::Exponential->new(0, 0),
            building     => [
                {
                    time        => 300,
                    count_built => 1,
                    power       => Isu7final::Exponential->new(288, 0),
                },
            ],
        },
    ),
    adding   => [],
    schedule => [
        {
            time        => 0,
            milli_isu   => Isu7final::Exponential->new(9996400000, 0),
            total_power => Isu7final::Exponential->new(0, 0),
        },
        {
            time        => 100,
            milli_isu   => Isu7final::Exponential->new(9996400000, 0),
            total_power => Isu7final::Exponential->new(16, 0),
        },
        {
            time        => 200,
            milli_isu   => Isu7final::Exponential->new(9996401600, 0),
            total_power => Isu7final::Exponential->new(72, 0),
        },
        {
            time        => 300,
            milli_isu   => Isu7final::Exponential->new(9996408800, 0),
            total_power => Isu7final::Exponential->new(360, 0),
        },
    ],
    on_sale  => bag(
        {
            item_id => 1,
            time    => 0,
        },
        {
            item_id => 2,
            time    => 0,
        },
    ),
});

done_testing;
