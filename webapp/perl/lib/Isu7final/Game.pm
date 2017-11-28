package Isu7final::Game;

use Mojo::Base 'Mojolicious::Plugin';
use Mojo::JSON qw(true false encode_json);
use Mojo::IOLoop;
use JSON::Types;
use Isu7final::Exponential;
use Isu7final::MItem;
use Math::BigInt lib => "GMP";

sub str2big {
    my $s = shift;

    return Math::BigInt->new($s // 0);
}

sub big2exp {
    my ($n) = @_;

    if (!$n) {
        return Isu7final::Exponential->new(0, 0);
    } elsif ($n->length <= 15) {
        return Isu7final::Exponential->new($n->numify, 0);
    } else {
        return Isu7final::Exponential->new(substr($n->bstr, 0, 15), $n->length - 15);
    }
}

sub calc_status {
    my ($current_time, $m_items, $addings, $buyings) = @_;

    # 1ミリ秒に生産できる椅子の単位をミリ椅子とする
    my $total_milli_isu = Math::BigInt->new;
    my $total_power     = Math::BigInt->new;

    my $item_power    = {}; # ItemID => Power
    my $item_price    = {}; # ItemID => Price
    my $item_on_sale  = {}; # ItemID => OnSale
    my $item_built    = {}; # ItemID => BuiltCount
    my $item_bought   = {}; # ItemID => CountBought
    my $item_building = {}; # ItemID => Buildings
    my $item_power0   = {}; # ItemID => currentTime における Power
    my $item_built0   = {}; # ItemID => currentTime における BuiltCount

    my $adding_at     = {}; # Time => currentTime より先の Adding
    my $buying_at     = {}; # Time => currentTime より先の Buying

    for my $item_id (keys %$m_items) {
        $item_power->{$item_id}    = Math::BigInt->new;
        $item_building->{$item_id} = [];
    }

    for my $a (@$addings) {
        # adding は adding.time に isu を増加させる
        if ($a->{time} <= $current_time) {
            $total_milli_isu->badd(str2big($a->{isu})->bmul(1000));
        } else {
            $adding_at->{$a->{time}} = {
                room_name => string $a->{room_name},
                time      => number $a->{time},
                isu       => string $a->{isu},
            };
        }
    }

    for my $b (@$buyings) {
        # buying は 即座に isu を消費し buying.time からアイテムの効果を発揮する
        $item_bought->{$b->{item_id}}++;
        my $m = $m_items->{$b->{item_id}};
        $total_milli_isu->bsub($m->get_price($b->{ordinal})->bmul(1000));

        if ($b->{time} <= $current_time) {
            $item_built->{$b->{item_id}}++;
            my $power = $m->get_power($item_bought->{$b->{item_id}});
            $total_milli_isu->badd($power->copy->bmul($current_time - $b->{time}));
            $total_power->badd($power);
            $item_power->{$b->{item_id}}->badd($power);
        } else {
            push @{$buying_at->{$b->{time}}}, $b;
        }
    }

    for my $m (values %$m_items) {
        $item_power0->{$m->{item_id}} = big2exp($item_power->{$m->{item_id}});
        $item_built0->{$m->{item_id}} = $item_built->{$m->{item_id}} // 0;
        my $price = $m->get_price(($item_bought->{$m->{item_id}} // 0) + 1);
        $item_price->{$m->{item_id}} = $price;
        if (0 <= $total_milli_isu->bcmp($price->copy->bmul(1000))) {
            $item_on_sale->{$m->{item_id}} = 0; # 0 は 時刻 currentTime で購入可能であることを表す
        }
    }

    my $schedule = [
        {
            time        => number $current_time,
            milli_isu   => big2exp($total_milli_isu),
            total_power => big2exp($total_power),
        },
    ];

    # currentTime から 1000 ミリ秒先までシミュレーションする
    for (my $t = $current_time + 1; $t <= $current_time + 1000; $t++) {
        $total_milli_isu->badd($total_power);
        my $updated = false;

        # 時刻 t で発生する adding を計算する
        if (exists $adding_at->{$t}) {
            my $a = $adding_at->{$t};
            $updated = true;
            $total_milli_isu->badd(str2big($a->{isu})->bmul(1000));
        }

        # 時刻 t で発生する buying を計算する
        if (exists $buying_at->{$t}) {
            $updated = true;
            my $updated_id = {};
            for my $b (@{$buying_at->{$t}}) {
                my $m = $m_items->{$b->{item_id}};
                $updated_id->{$b->{item_id}} = true;
                $item_power->{$b->{item_id}}->badd($m->get_power($b->{ordinal}));
                $item_built->{$b->{item_id}}++;
                $total_power->badd($m->get_power($b->{ordinal}));
            }
            for my $id (keys %$updated_id) {
                push @{$item_building->{$id}}, {
                    time        => number $t,
                    count_built => number $item_built->{$id},
                    power       => big2exp($item_power->{$id}),
                };
            }
        }

        if ($updated) {
            push @$schedule, {
                time        => number $t,
                milli_isu   => big2exp($total_milli_isu),
                total_power => big2exp($total_power),
            };
        }

        # 時刻 t で購入可能になったアイテムを記録する
        for my $item_id (keys %$m_items) {
            if (exists $item_on_sale->{$item_id}) {
                next;
            }
            if (0 <= $total_milli_isu->bcmp($item_price->{$item_id}->copy->bmul(1000))) {
                $item_on_sale->{$item_id} = $t;
            }
        }
    }

    my $gs_adding = [ values %$adding_at ];


    my $gs_items = [ map {
        my $item_id = $_;

        +{
            item_id      => number $item_id,
            count_bought => number ($item_bought->{$item_id} // 0),
            count_built  => number $item_built0->{$item_id},
            next_price   => big2exp($item_price->{$item_id}),
            power        => $item_power0->{$item_id},
            building     => $item_building->{$item_id},
        }
    } keys %$m_items ];

    my $gs_on_sale = [];
    for my $item_id (keys %$item_on_sale) {
        my $t = $item_on_sale->{$item_id};
        push @$gs_on_sale, {
            item_id => number $item_id,
            time    => number $t,
        };
    }

    return {
        time     => 0,
        adding   => $gs_adding,
        schedule => $schedule,
        items    => $gs_items,
        on_sale  => $gs_on_sale,
    };
}

sub register {
    my ($self, $app, $conf) = @_;

    $app->helper(add_isu => sub {
        my ($c, $room_name, $req_isu, $req_time) = @_;

        my $db = $c->mysql->db; # get connection
        my $tx = $db->begin;    # start transaction

        my $current_time = $c->update_room_time($db, $room_name, $req_time);
        if (!$current_time) {
            return false;
        }

        eval { $db->query("INSERT INTO adding(room_name, time, isu) VALUES (?, ?, '0') ON DUPLICATE KEY UPDATE isu=isu", $room_name, $req_time) };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }

        my $isu_str = eval { $db->query("SELECT isu FROM adding WHERE room_name = ? AND time = ? FOR UPDATE", $room_name, $req_time)->array->[0] };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }

        my $isu = str2big($isu_str);

        $isu->badd($req_isu);

        eval { $db->query("UPDATE adding SET isu = ? WHERE room_name = ? AND time = ?", $isu->bstr, $room_name, $req_time) };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }

        $tx->commit;

        return true;
    });

    $app->helper(buy_item => sub {
        my ($c, $room_name, $item_id, $count_bought, $req_time) = @_;

        $count_bought //= 0;

        my $db = $c->mysql->db; # get connection
        my $tx = $db->begin;    # start transaction

        my $current_time = $c->update_room_time($db, $room_name, $req_time);
        if (!$current_time) {
            return false;
        }

        my $count_buying = eval { $db->query("SELECT COUNT(*) FROM buying WHERE room_name = ? AND item_id = ?", $room_name, $item_id)->array->[0] };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }

        if ($count_buying != $count_bought) {
            $c->app->log->error($room_name, $item_id, $count_bought + 1, " is already bought");
            return false;
        }

        my $total_milli_isu = Math::BigInt->new;
        my $addings = eval { $db->query("SELECT isu FROM adding WHERE room_name = ? AND time <= ?", $room_name, $req_time)->hashes->to_array };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }
        for my $a (@$addings) {
            $total_milli_isu->badd(str2big($a->{isu})->bmul(1000));
        }

        my $buyings = eval { $db->query("SELECT item_id, ordinal, time FROM buying WHERE room_name = ?", $room_name)->hashes->to_array };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }
        for my $b (@$buyings) {
            my $m_item = Isu7final::MItem->new($db->query("SELECT * FROM m_item WHERE item_id = ?", $b->{item_id})->hash);
            my $cost   = $m_item->get_price($b->{ordinal})->bmul(1000);

            $total_milli_isu->bsub($cost);

            if ($b->{time} <= $req_time) {
                my $gain = $m_item->get_power($b->{ordinal})->bmul($req_time - $b->{time});

                $total_milli_isu->badd($gain);
            }
        }

        my $item = eval { Isu7final::MItem->new($db->query("SELECT * FROM m_item WHERE item_id = ?", $item_id)->hash) };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }
        my $need = $item->get_price($count_bought + 1)->bmul(1000);
        if ($total_milli_isu->bcmp($need) < 0) {
            $c->app->log->error("not enough");
            return false;
        }

        eval { $db->query("INSERT INTO buying(room_name, item_id, ordinal, time) VALUES(?, ?, ?, ?)", $room_name, $item_id, $count_bought + 1, $req_time) };
        if (my $e = $@) {
            $c->app->log->error($e);
            return false;
        }

        $tx->commit;

        return true;
    });

    $app->helper(update_room_time => sub {
        # 部屋のロックを取りタイムスタンプを更新する
        #
        # トランザクション開始後この関数を呼ぶ前にクエリを投げると、
        # そのトランザクション中の通常のSELECTクエリが返す結果がロック取得前の
        # 状態になることに注意 (keyword: MVCC, repeatable read).

        my ($c, $db, $room_name, $req_time) = @_;

        # See page 13 and 17 in https://www.slideshare.net/ichirin2501/insert-51938787
        eval { $db->query("INSERT INTO room_time(room_name, time) VALUES (?, 0) ON DUPLICATE KEY UPDATE time = time", $room_name) };
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        my $room_time = eval{ $db->query("SELECT time FROM room_time WHERE room_name = ? FOR UPDATE", $room_name)->array->[0] };
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        my $current_time = eval { $db->query("SELECT floor(unix_timestamp(current_timestamp(3))*1000)")->array->[0] };
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        if ($room_time > $current_time) {
            $c->app->log->error("room time is future");
            return;
        }

        if ($req_time != 0) {
            if ($req_time < $current_time) {
                $c->app->log->error("reqTime is past");
                return;
            }
        }

        eval { $db->query("UPDATE room_time SET time = ? WHERE room_name = ?", $current_time, $room_name) };
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        return $current_time;
    });

    $app->helper(get_current_time => sub {
        my ($c, $room_name) = @_;

        my $db = $c->mysql->db;

        my $current_time = eval { $db->query("SELECT floor(unix_timestamp(current_timestamp(3))*1000)")->array->[0] };
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        return $current_time;
    });

    $app->helper(get_status => sub {
        my ($c, $room_name) = @_;

        my $db = $c->mysql->db; # get connection
        my $tx = $db->begin; # start transaction

        my $current_time = $c->update_room_time($db, $room_name, 0);

        return if (!$current_time);

        my $m_items = eval {
            my $result = {};
            $db->query("SELECT * FROM m_item")->hashes->each(sub { $result->{$_->{item_id}} = Isu7final::MItem->new(%$_) });
            $result;
        };
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        my $addings = eval { $db->query("SELECT time, isu FROM adding WHERE room_name = ?", $room_name)->hashes->to_array } ;
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        my $buyings = eval { $db->query("SELECT item_id, ordinal, time FROM buying WHERE room_name = ?", $room_name)->hashes->to_array };
        if (my $e = $@) {
            $c->app->log->error($e);
            return;
        }

        $tx->commit;

        my $status = calc_status($current_time, $m_items, $addings, $buyings);

        return if (!$status);

        # calc_statusに時間がかかる可能性があるので タイムスタンプを取得し直す
        my $latest_time = $c->get_current_time;

        return if (!$latest_time);

        $status->{time} = number $latest_time;

        return $status;
    });

    $app->helper(serve_game_conn => sub {
        my ($c, $room_name) = @_;

        $c->app->log->debug(sprintf "%s:%d %s %s", $c->tx->remote_address, $c->tx->remote_port, "serve_game_conn", $room_name);

        my $get_status = sub {
            my $status = $c->get_status($room_name);
            $c->send({ json => $status });
        };

        my $ticker; $ticker = Mojo::IOLoop->recurring(0.5 => sub {
            eval {
                my $status = $c->get_status($room_name);
                $c->send({ json => $status });
            };
            Mojo::IOLoop->remove($ticker) if ($@);
        });

        $c->on(json => sub {
            my ($c, $req) = @_;
            $c->app->log->debug(encode_json $req);

            my $success = false;
            if ($req->{action} eq "addIsu") {
                $success = $c->add_isu($room_name, str2big($req->{isu}), $req->{time});
            } elsif ($req->{action} eq "buyItem") {
                $success = $c->buy_item($room_name, $req->{item_id}, $req->{count_bought},  $req->{time});
            } else {
                $c->app->log->warn("Invalid Action");
                return;
            }

            if ($success) {
                # GameResponse を返却する前に 反映済みの GameStatus を返す
                my $status = $c->get_status($room_name);
                $c->send({ json => $status });
            }

            $c->send({ json => {
                request_id => number $req->{request_id},
                is_success => bool   $success,
            }});
        });

        $c->on(finish => sub {
            Mojo::IOLoop->remove($ticker);
        });

        $get_status->(); # recurring は待ってから動くので
    });
}

1;
