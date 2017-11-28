use Test::More;
use Test::Deep;
use Isu7final::Game;
use Isu7final::MItem;

my $x = Isu7final::MItem->new(
    item_id => 1,
    power1  => 0,
    power2  => 1,
    power3  => 0,
    power4  => 10,
    price1  => 0,
    price2  => 1,
    price3  => 0,
    price4  => 10,
);

my $m_items     = { 1 => $x };
my $initial_isu = 10;
my $addings     = [{ time => 0, isu => $initial_isu }];
my $buyings     = [{ item_id => 1, ordinal => 1, time => 100 }];

my $got = Isu7final::Game::calc_status(0, $m_items, $addings, $buyings);

cmp_deeply $got, superhashof({
    time     => 0,
    items    => [{
        item_id      => 1,
        count_bought => 1,
        count_built  => 0,
        next_price   => Isu7final::Exponential->new(10, 0),
        power        => Isu7final::Exponential->new(0, 0),
        building     => [{
            time        => 100,
            count_built => 1,
            power       => Isu7final::Exponential->new(10, 0),
        }],
    }],
    adding   => [],
    schedule => [
        {
            time        => 0,
            milli_isu   => Isu7final::Exponential->new(0, 0),
            total_power => Isu7final::Exponential->new(0, 0),
        },
        {
            time        => 100,
            milli_isu   => Isu7final::Exponential->new(0, 0),
            total_power => Isu7final::Exponential->new(10, 0),
        },
    ],
    on_sale  => [],
});

done_testing;
