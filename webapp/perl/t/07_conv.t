use Test::More;
use Test::Deep;
use Isu7final::Game;
use Isu7final::Exponential;
use Math::BigInt;

cmp_deeply Isu7final::Game::big2exp(Math::BigInt->new(0)), Isu7final::Exponential->new(0, 0);
cmp_deeply Isu7final::Game::big2exp(Math::BigInt->new(1234)), Isu7final::Exponential->new(1234, 0);
cmp_deeply Isu7final::Game::big2exp(Math::BigInt->new(11111111111111000000)), Isu7final::Exponential->new(111111111111110, 5);

done_testing;
