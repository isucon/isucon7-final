<?php

declare(strict_types=1);

use PHPUnit\Framework\TestCase;

class Isu7Test extends TestCase
{
    public function testStatusEmpty()
    {
        $mItems = [];
        $addings = [];
        $buyings = [];
        
        $s = calcStatus(0, $mItems, $addings, $buyings);

        $this->assertNotNull($s);
        $this->assertEmpty($s->adding);
        $this->assertCount(1, $s->schedule);
        $this->assertEmpty($s->onSale);

        $this->assertEquals(0, $s->schedule[0]->time);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[0]->milliIsu);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[0]->totalPower);
    }

    // 椅子が増える
    public function testStatusAdd()
    {
        $mItems = [];
        $addings = [
            new Adding('', 100, "1"),
            new Adding('', 200, "2"),
            new Adding('', 300, "1234567890123456789"),
        ];
        $buyings = [];

        $s = calcStatus(0, $mItems, $addings, $buyings);
        $this->assertNotNull($s);
        $this->assertCount(3, $s->adding);
        $this->assertCount(4, $s->schedule);

        $this->assertEquals(0, $s->schedule[0]->time);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[0]->milliIsu);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[0]->totalPower);

        $this->assertEquals(100, $s->schedule[1]->time);
        $this->assertEquals(new Exponential(1000, 0), $s->schedule[1]->milliIsu);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[1]->totalPower);

        $this->assertEquals(200, $s->schedule[2]->time);
        $this->assertEquals(new Exponential(3000, 0), $s->schedule[2]->milliIsu);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[2]->totalPower);

        $this->assertEquals(300, $s->schedule[3]->time);
        $this->assertEquals(new Exponential(123456789012345, 7), $s->schedule[3]->milliIsu);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[3]->totalPower);

        $s = calcStatus(500, $mItems, $addings, $buyings);
        $this->assertNotNull($s);
        $this->assertCount(0, $s->adding);
        $this->assertCount(1, $s->schedule);

        $this->assertEquals(500, $s->schedule[0]->time);
        $this->assertEquals(new Exponential(123456789012345, 7), $s->schedule[0]->milliIsu);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[0]->totalPower);
    }

    // 試しに１個買う
    public function testStatusBuySingle()
    {
        $x = new mItem(1, 0, 1, 0, 10, 0, 1, 0, 10);
        $mItems = [1 => $x];
        $initialIsu = "10";
        $addings = [
            new Adding('', 0, $initialIsu)
        ];
        $buyings = [
            new Buying('', 1, 1, 100),
        ];
        $s = calcStatus(0, $mItems, $addings, $buyings);
        $this->assertNotNull($s);
        $this->assertCount(0, $s->adding);
        $this->assertCount(2, $s->schedule);
        $this->assertCount(1, $s->items);

        $this->assertEquals(0, $s->schedule[0]->time);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[0]->milliIsu);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[0]->totalPower);

        $this->assertEquals(100, $s->schedule[1]->time);
        $this->assertEquals(new Exponential(0, 0), $s->schedule[1]->milliIsu);
        $this->assertEquals(new Exponential(10, 0), $s->schedule[1]->totalPower);
    }

    // 購入時間を見ます
    public function testOnSale()
    {
        // power: (0x+1)*1^(0x+1)
        // price: (0x+1)*1^(0x+1)
           $x = new mItem(1, 0, 1, 0, 1, 0, 1, 0, 1);
           $mItems = [1 => $x];
           $addings = [
               new Adding($roomName = '', $time = 0, $isu = "1")
               ];
           $buyings = [
               new Buying($roomname = '', $itemId = 1, $ordinal = 1, $time = 0)
               ];

           $s = calcStatus(1, $mItems, $addings, $buyings);
           $this->assertNotNull($s);
           $this->assertCount(0, $s->adding);
           $this->assertCount(1, $s->schedule);
           $this->assertCount(1, $s->onSale);
           $this->assertCount(1, $s->items);

           $this->assertEquals(1, $s->schedule[0]->time);
           $this->assertEquals(new OnSale($itemId = 1, $time = 1000), $s->onSale[0]);

           $this->assertEquals($s->items[0]->countBought, 1);
           $this->assertEquals($s->items[0]->power, new Exponential(1, 0));
           $this->assertEquals($s->items[0]->countBuilt, 1);
           $this->assertEquals($s->items[0]->nextPrice, new Exponential(1, 0));
    }

    public function testStatusBuy()
    {
           $x = new mItem($itemId = 1, $power1 = 1, $power2 = 1, $power3 = 3, $power4 = 2, $price1 = 1, $price2 = 1, $price3 = 7, $price4 = 6);
           $y = new mItem($itemId = 2, $power1 = 1, $power2 = 1, $power3 = 7, $power4 = 6, $price1 = 1, $price2 = 1, $price3 = 3, $price4 = 2);

           $mItems = [1 => $x, 2 => $y];
           $initialIsu = "10000000";
           $addings = [
               new Adding($roomName = '', $time = 0, $isu = $initialIsu),
           ];
           $buyings = [
               new Buying($roomName = '', $itemId = 1, $ordinal = 1, $time = 100),
               new Buying($roomName = '', $itemId = 1, $ordinal = 2, $time = 200),
               new Buying($roomName = '', $itemId = 2, $ordinal = 1, $time = 300),
               new Buying($roomName = '', $itemId = 2, $ordinal = 2, $time = 2001),
           ];

           $s = calcStatus(0, $mItems, $addings, $buyings);
           $this->assertNotNull($s);
           $this->assertCount(0, $s->adding);
           $this->assertCount(4, $s->schedule);
           $this->assertCount(2, $s->onSale);
           $this->assertCount(2, $s->items);

           $totalPower = gmp_init(0);
           $milliIsu = gmp_mul($initialIsu, gmp_init(1000));
           $milliIsu = gmp_sub($milliIsu, gmp_mul($x->getPrice(1), gmp_init(1000, 10)));
           $milliIsu = gmp_sub($milliIsu, gmp_mul($x->getPrice(2), gmp_init(1000, 10)));
           $milliIsu = gmp_sub($milliIsu, gmp_mul($y->getPrice(1), gmp_init(1000, 10)));
           $milliIsu = gmp_sub($milliIsu, gmp_mul($y->getPrice(2), gmp_init(1000, 10)));

           // 0sec
           $this->assertEquals(0, $s->schedule[0]->time);
           $this->assertEquals(big2exp($milliIsu), $s->schedule[0]->milliIsu);
           $this->assertEquals(big2exp($totalPower), $s->schedule[0]->totalPower);

           // 0.1sec
           $totalPower = gmp_add($totalPower, $x->getPower(1));
           $this->assertEquals(100, $s->schedule[1]->time);
           $this->assertEquals(big2exp($milliIsu), $s->schedule[1]->milliIsu);
           $this->assertEquals(big2exp($totalPower), $s->schedule[1]->totalPower);

           // 0.2sec
           $milliIsu = gmp_add($milliIsu, gmp_mul($totalPower, gmp_init(100)));
           $totalPower = gmp_add($totalPower, $x->getPower(2));
           $this->assertEquals(200, $s->schedule[2]->time);
           $this->assertEquals(big2exp($milliIsu), $s->schedule[2]->milliIsu);
           $this->assertEquals(big2exp($totalPower), $s->schedule[2]->totalPower);

           // 0.3sec
           $milliIsu = gmp_add($milliIsu, gmp_mul($totalPower, 100));
           $totalPower = gmp_add($totalPower, $y->getPower(1));
           $this->assertEquals(300, $s->schedule[3]->time);
           $this->assertEquals(big2exp($milliIsu), $s->schedule[3]->milliIsu);
           $this->assertEquals(big2exp($totalPower), $s->schedule[3]->totalPower);
           
           // OnSale
           $this->assertTrue(in_array(new OnSale($itemId = 1, $time = 0), $s->onSale));
           $this->assertTrue(in_array(new OnSale($itemId = 2, $time = 0), $s->onSale));
    }

    public function testMItem()
    {
           $item = new mItem($itemId = 1, $power1 = 1, $power2 = 2, $power3 = 2, $power4 = 3, $price1 = 5, $price2 = 4, $price3 = 3, $price4 = 2);
           $this->assertEquals($item->getPower(1), gmp_init(81));
           $this->assertEquals($item->getPrice(1), gmp_init(2048));
    }

    public function testConv()
    {
           $this->assertEquals(new Exponential(0, 0), big2exp(str2big("0")));
           $this->assertEquals(new Exponential(1234, 0), big2exp(str2big("1234")));
           $this->assertEquals(new Exponential(111111111111110, 5), big2exp(str2big("11111111111111000000")));
    }
}
