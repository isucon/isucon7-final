use Test::More;
use Test::Deep;
use Isu7final::Game;
use Isu7final::MItem;

my $x = Isu7final::MItem->new(
    item_id => 1,
    power1  => 0, power2  => 1, power3  => 0, power4  => 1, # power: (0x+1)*1^(0x+1)
    price1  => 0, price2  => 1, price3  => 0, price4  => 1, # price: (0x+1)*1^(0x+1)
);

my $m_items     = { 1 => $x };
my $addings     = [{ time => 0, isu => 1 }];
my $buyings     = [{ item_id => 1, ordinal => 1, time => 0 }];

my $got = Isu7final::Game::calc_status(1, $m_items, $addings, $buyings);

cmp_deeply $got, superhashof({
    time     => 0,
    items    => [{
        item_id      => 1,
        count_bought => 1,
        count_built  => 1,
        next_price   => Isu7final::Exponential->new(1, 0),
        power        => Isu7final::Exponential->new(1, 0),
        building     => [],
    }],
    adding   => [],
    schedule => [
        {
            time        => 1,
            milli_isu   => Isu7final::Exponential->new(1, 0),
            total_power => Isu7final::Exponential->new(1, 0),
        },
    ],
    on_sale  => [{
        item_id => 1,
        time    => 1000,
    }],
});

done_testing;
