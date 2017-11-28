use Test::More;
use Math::BigInt;
use Isu7final::MItem;

my $m_item = Isu7final::MItem->new(
    item_id => 1,
    power1 => 1, power2  => 2, power3  => 2, power4  => 3,
    price1 => 5, price2  => 4, price3  => 3, price4  => 2,
);

is $m_item->get_power(1), 81;
is $m_item->get_price(1), 2048;

done_testing;
