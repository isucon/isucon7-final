<?php

declare(strict_types=1);

use React\Promise\Deferred;
use Monolog\Logger;
use Monolog\Handler\StreamHandler;

require_once __DIR__ . '/vendor/autoload.php';

$loop = React\EventLoop\Factory::create();

/**
 * @return Monolog\Logger
 */
function getLogger()
{
    static $log = null;
    if ($log) {
        return $log;
    }

    $log = new Logger('isulog');
    $log->pushHandler(new StreamHandler('php://stdout'));
    return $log;
}

interface Hydratable
{
    public function hydrate(array $assoc);
}

/**
 * @param GMP $n
 * @param GMP $p
 * @return GMP
 */
function gmp_exp(GMP $n, GMP $p): GMP
{
    if ($p == 0) {
        return gmp_init(0, 10);
    }
    if ($p == 1) {
        return $n;
    }

    $m = gmp_exp($n, gmp_div($p, 2));
    $m = gmp_mul($m, $m);
    if (gmp_mod($p, 2) == 1) {
        $m = gmp_mul($m, $n);
    }
    return $m;
}

/**
 * 10進数の指数表記に使うデータ。JSONでは [仮数部, 指数部] という2要素配列になる。
 */
class Exponential implements JsonSerializable
{
    // $mantissa * 10 ^ $exponent

    /** @var int */
    public $mantissa;

    /** @var int */
    public $exponent;

    public function __construct($mantissa, $exponent)
    {
        $this->mantissa = $mantissa;
        $this->exponent = $exponent;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        return [$this->mantissa, $this->exponent];
    }
}

class GameRequest implements Hydratable
{
    /** @var int */
    public $requestId;

    /** @var string */
    public $action;

    /** @var int */
    public $time;

    // for addIsu

    /** @var string */
    public $isu;

    // for buyItem

    /** @var int */
    public $itemId;

    /** @var int */
    public $countBought;

    public function __construct()
    {
    }

    public function hydrate(array $assoc)
    {
        $this->requestId   = (int)($assoc['request_id'] ?? 0);
        $this->action      = $assoc['action'] ?? '';
        $this->time        = (int)($assoc['time'] ?? 0);
        $this->isu         = ($assoc['isu'] ?? '');
        $this->itemId      = (int)($assoc['item_id'] ?? 0);
        $this->countBought = (int)($assoc['count_bought'] ?? 0);
    }
}

class GameResponse implements JsonSerializable
{
    /** @var string */
    public $requestId;

    /** @var bool */
    public $isSuccess;

    public function __construct(string $requestId, bool $isSuccess)
    {
        $this->requestId = $requestId;
        $this->isSuccess = $isSuccess;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        return [
            "request_id" => $this->requestId,
            "is_success" => $this->isSuccess
        ];
    }
}

class Adding implements JsonSerializable, Hydratable
{
    /** @var string */
    public $roomName;

    /** @var int */
    public $time;

    /** @var string */
    public $isu;

    public function __construct(string $roomName = '', int $time = 0, string $isu = '')
    {
        $this->roomName = $roomName;
        $this->time     = $time;
        $this->isu      = $isu;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        return [
            "room_name" => $this->roomName,
            "time"      => $this->time,
            "isu"       => $this->isu,
        ];
    }

    public function hydrate(array $assoc)
    {
        $this->roomName = $assoc['room_name'] ?? '';
        $this->time     = (int)($assoc['time'] ?? 0);
        $this->isu      = $assoc['isu'] ?? '';
    }
}

class Buying implements Hydratable
{
    /** @var string */
    public $roomName;

    /** @var int */
    public $itemId;

    /** @var int */
    public $ordinal;

    /** @var int */
    public $time;

    public function __construct(string $roomName = '', int $itemId = 0, int $ordinal = 0, int $time = 0)
    {
        $this->roomName = $roomName;
        $this->itemId   = $itemId;
        $this->ordinal  = $ordinal;
        $this->time     = $time;
    }

    public function hydrate(array $assoc)
    {
        $this->roomName = $assoc['room_name'] ?? '';
        $this->itemId   = (int)($assoc['item_id'] ?? 0);
        $this->ordinal  = (int)($assoc['ordinal'] ?? 0);
        $this->time     = (int)($assoc['time'] ?? 0);
    }
}

class Schedule implements JsonSerializable
{
    /** @var int */
    public $time;

    /** @var Exponential */
    public $milliIsu;

    /** @var Exponential */
    public $totalPower;

    public function __construct(int $time = 0, Exponential $milliIsu = null, Exponential $totalPower = null)
    {
        $this->time       = $time;
        $this->milliIsu   = $milliIsu;
        $this->totalPower = $totalPower;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        return [
            "time"        => $this->time,
            "milli_isu"   => $this->milliIsu->jsonSerialize(),
            "total_power" => $this->totalPower->jsonSerialize(),
        ];
    }
}

class Item implements JsonSerializable
{
    /** @var int */
    public $itemId;

    /** @var int */
    public $countBought;

    /** @var int */
    public $countBuilt;

    /** @var Exponential */
    public $nextPrice;

    /** @var Exponential */
    public $power;

    /** @var Building[] */
    public $building;

    public function __construct(int $itemId, int $countBought, int $countBuilt, Exponential $nextPrice, Exponential $power, array $building)
    {
        $this->itemId = $itemId;
        $this->countBought = $countBought;
        $this->countBuilt = $countBuilt;
        $this->nextPrice = $nextPrice;
        $this->power = $power;
        $this->building = $building;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        $building = [];
        foreach ($this->building as $b) {
            $building[] = $b->jsonSerialize();
        }

        return [
            "item_id"      => $this->itemId,
            "count_bought" => $this->countBought,
            "count_built"  => $this->countBuilt,
            "next_price"   => $this->nextPrice->jsonSerialize(),
            "power"        => $this->power->jsonSerialize(),
            "building"     => $building,
        ];
    }
}

class OnSale implements JsonSerializable
{
    /** @var int */
    public $itemId;

    /** @var int */
    public $time;

    public function __construct(int $itemId = 0, int $time = 0)
    {
        $this->itemId = $itemId;
        $this->time   = $time;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        return [
            "item_id" => $this->itemId,
            "time"    => $this->time,
        ];
    }
}

class Building implements JsonSerializable
{
    /** @var int */
    public $time;

    /** @var int */
    public $countBuilt;

    /** @var Exponential */
    public $power;

    public function __construct(int $time = 0, int $countBuilt = 0, Exponential $power = null)
    {
        $this->time       = $time;
        $this->countBuilt = $countBuilt;
        $this->power      = $power;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        return [
            "time"        => $this->time,
            "count_built" => $this->countBuilt,
            "power"       => $this->power,
        ];
    }
}

class GameStatus implements JsonSerializable
{
    /** @var int */
    public $time;

    /** @var Adding[] */
    public $adding;

    /** @var Schedule[] */
    public $schedule;

    /** @var Item[] */
    public $items;

    /** @var OnSale[] */
    public $onSale;

    public function __construct(int $time, array $adding, array $schedule, array $items, array $onSale)
    {
        $this->time     = $time;
        $this->adding   = $adding;
        $this->schedule = $schedule;
        $this->items    = $items;
        $this->onSale   = $onSale;
    }

    /**
     * {@inheritdoc}
     */
    public function jsonSerialize()
    {
        $adding = [];
        foreach ($this->adding as $a) {
            $adding[] = $a->jsonSerialize();
        }

        $schedule = [];
        foreach ($this->schedule as $s) {
            $schedule[] = $s->jsonSerialize();
        }

        $items = [];
        foreach ($this->items as $i) {
            $items[] = $i->jsonSerialize();
        }

        $onSale = [];
        foreach ($this->onSale as $s) {
            $onSale[] = $s->jsonSerialize();
        }

        return [
            "time"     => $this->time,
            "adding"   => $adding,
            "schedule" => $schedule,
            "items"    => $items,
            "on_sale"  => $onSale,
        ];
    }
}

class MItem implements Hydratable
{
    /** @var int */
    public $itemId;

    /** @var int */
    public $power1;

    /** @var int */
    public $power2;

    /** @var int */
    public $power3;

    /** @var int */
    public $power4;

    /** @var int */
    public $price1;

    /** @var int */
    public $price2;

    /** @var int */
    public $price3;

    /** @var int */
    public $price4;

    public function __construct($itemId = 0, $power1 = 0, $power2 = 0, $power3 = 0, $power4 = 0, $price1 = 0, $price2 = 0, $price3 = 0, $price4 = 0)
    {
        $this->itemId = $itemId;
        $this->power1 = $power1;
        $this->power2 = $power2;
        $this->power3 = $power3;
        $this->power4 = $power4;
        $this->price1 = $price1;
        $this->price2 = $price2;
        $this->price3 = $price3;
        $this->price4 = $price4;
    }

    /**
     * @param int $count
     * @return GMP
     */
    public function getPower(int $count)
    {
        // power(x):=(cx+1)*d^(ax+b)
        $a = gmp_init($this->power1, 10);
        $b = gmp_init($this->power2, 10);
        $c = gmp_init($this->power3, 10);
        $d = gmp_init($this->power4, 10);
        $x = gmp_init($count, 10);

        $s = gmp_add(gmp_mul($c, $x), 1);
        $t = gmp_exp($d, gmp_add(gmp_mul($a, $x), $b));
        return gmp_mul($s, $t);
    }

    /**
     * @param int $count
     * @return GMP
     */
    public function getPrice(int $count): GMP
    {
        // price(x):=(cx+1)*d^(ax+b)
        $a = gmp_init($this->price1, 10);
        $b = gmp_init($this->price2, 10);
        $c = gmp_init($this->price3, 10);
        $d = gmp_init($this->price4, 10);
        $x = gmp_init($count, 10);

        $s = gmp_add(gmp_mul($c, $x), 1);
        $t = gmp_exp($d, gmp_add(gmp_mul($a, $x), $b));
        return gmp_mul($s, $t);
    }

    public function hydrate(array $assoc)
    {
        $this->itemId = (int)($assoc['item_id'] ?? 0);
        $this->power1 = (int)($assoc['power1'] ?? 0);
        $this->power2 = (int)($assoc['power2'] ?? 0);
        $this->power3 = (int)($assoc['power3'] ?? 0);
        $this->power4 = (int)($assoc['power4'] ?? 0);
        $this->price1 = (int)($assoc['price1'] ?? 0);
        $this->price2 = (int)($assoc['price2'] ?? 0);
        $this->price3 = (int)($assoc['price3'] ?? 0);
        $this->price4 = (int)($assoc['price4'] ?? 0);
    }
}

/**
 * @return React\MySQL\Connection
 */
function getDbConnection()
{
    global $loop;

    $host = getenv('ISU_DB_HOST') ?: 'localhost';
    $port = getenv('ISU_DB_PORT') ?: '3306';
    $user = getenv('ISU_DB_USER') ?: 'root';
    $password = getenv('ISU_DB_PASSWORD') ?: '';

    $param = [
        'dbname' => 'isudb',
        'host' => $host,
        'port'   => $port,
        'user'   => $user,
        'passwd' => $password,
    ];
    $connection = new React\MySQL\Connection($loop, $param);
    $connection->connect(function () {
    });
    return $connection;
}

/**
 * @param GMP n
 * @return Exponential
 */
function big2exp(GMP $n)
{
    $s = gmp_strval($n);
    if (strlen($s) <= 15) {
        return new Exponential(gmp_intval($n), 0);
    }
    $t = substr($s, 0, 15);
    return new Exponential((int)$t, strlen($s) - 15);
}

/**
 * @return GMP
 */
function str2big(string $s)
{
    return gmp_init($s, 10);
}

/**
 * @return React\Promise\Promise
 */
function getCurrentTime()
{
    $deferred = new React\Promise\Deferred();
    getDbConnection()->query(
        "SELECT floor(unix_timestamp(current_timestamp(3))*1000)",
        function ($command, $conn) use ($deferred) {
            $deferred->resolve((int)reset($command->resultRows[0]));
            $conn->close();
        }
    );
    return $deferred->promise();
}

/**
 * 部屋のロックを取りタイムスタンプを更新する
 *
 * トランザクション開始後この関数を呼ぶ前にクエリを投げると、
 * そのトランザクション中の通常のSELECTクエリが返す結果がロック取得前の
 * 状態になることに注意 (keyword: MVCC, repeatable read).
 */
function updateRoomTime(React\MySQL\Connection $db, React\Promise\Promise $promise, string $roomName, int $reqTime)
{
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $reqTime, $roomName) {
            $db->query(
                "INSERT INTO room_time(room_name, time) VALUES ('$roomName', 0) ON DUPLICATE KEY UPDATE time = time",
                function ($command, $conn) use ($deferred) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getSql() . "\n" . $command->getError());
                    } else {
                        $deferred->resolve(null);
                    }
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $reqTime, $roomName) {
            $db->query(
                "SELECT time FROM room_time WHERE room_name = '$roomName' FOR UPDATE",
                function ($command, $conn) use ($deferred, $reqTime) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getSql() . "\n" . $command->getError());
                        return ;
                    }
                    $roomTime = $command->resultRows[0]['time'];
                    $deferred->resolve($roomTime);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($roomTime) use ($db, $deferred, $reqTime, $roomName) {
            $db->query(
                "SELECT floor(unix_timestamp(current_timestamp(3))*1000)",
                function ($command, $conn) use ($deferred, $reqTime, $roomTime) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getSql() . "\n" . $command->getError());
                        return ;
                    }
                    $currentTime = (int)reset($command->resultRows[0]);
                    if ($roomTime > $currentTime) {
                        $deferred->reject("room time is future");
                        return ;
                    }
                    if ($reqTime != 0 && $reqTime < $currentTime) {
                        $deferred->reject("reqTime is past");
                        return ;
                    }
                    $deferred->resolve($currentTime);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($currentTime) use ($db, $deferred, $reqTime, $roomName) {
            $db->query(
                "UPDATE room_time SET time = $currentTime WHERE room_name = '$roomName'",
                function ($command, $conn) use ($deferred, $currentTime) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getSql() . "\n" . $command->getError());
                    } else {
                        $deferred->resolve($currentTime);
                    }
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    return $deferred->promise();
}

/**
 * @param string $roomName
 * @param GMP $reqIsu
 * @param int $reqTime
 * @return React\Promise\Promise
 */
function addIsu(string $roomName, GMP $reqIsu, int $reqTime)
{
    $db = getDbConnection();

    $deferred = new React\Promise\Deferred();
    $db->query('BEGIN', function ($command, $conn) use ($deferred) {
        $deferred->resolve(null);
    });
    $promise = updateRoomTime($db, $deferred->promise(), $roomName, $reqTime);

    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $reqIsu, $reqTime) {
            $db->query(
                "INSERT INTO adding(room_name, time, isu) VALUES ('$roomName', '$reqTime', '0') ON DUPLICATE KEY UPDATE isu=isu",
                function ($command, $conn) use ($deferred) {
                    $deferred->resolve(null);
                }
            );
            return $deferred->promise();
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $curIsu = "0";
    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, &$curIsu, $reqTime) {
            $db->query(
                "SELECT isu FROM adding WHERE room_name = '$roomName' AND time = '$reqTime' FOR UPDATE",
                function ($command, $conn) use ($deferred, &$curIsu) {
                    if (isset($command->resultRows[0]['isu'])) {
                        $curIsu = $command->resultRows[0]['isu'];
                    }
                    $deferred->resolve(null);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );


    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $reqIsu, &$curIsu, $reqTime) {
            $newIsu = gmp_strval(gmp_add($curIsu, $reqIsu));
            $db->query(
                "UPDATE adding SET isu = '$newIsu' WHERE room_name = '$roomName' AND time = $reqTime",
                function ($command, $conn) use ($deferred) {
                    $deferred->resolve(null);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $reqIsu, $reqTime) {
            $db->query(
                "COMMIT",
                function ($command, $conn) use ($deferred) {
                    $deferred->resolve(new GameResponse('', true));
                    $conn->close();
                }
            );
        },
        function ($err) use ($deferred, $db) {
            $db->query("ROLLBACK", function ($_, $conn) use ($deferred, $err) {
                $conn->close();
                getLogger()->error(__METHOD__ . "(" . __LINE__ . "): " . $err);
                $deferred->reject(new GameResponse('', false));
            });
        }
    );

    return $deferred->promise();
}

function buyItem(string $roomName, int $itemId, int $countBought, int $reqTime)
{
    $db = getDbConnection();
    $deferred = new React\Promise\Deferred();
    $db->query('BEGIN', function ($command, $conn) use ($deferred) {
        $deferred->resolve(null);
    });

    $currentTime = 0;
    $promise = updateRoomTime($db, $deferred->promise(), $roomName, 0);
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($time) use (&$currentTime, $deferred) {
            $currentTime = $time;
            $deferred->resolve(null);
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $itemId, $countBought, $reqTime) {
            $db->query(
                "SELECT COUNT(*) FROM buying WHERE room_name = '$roomName' AND item_id = '$itemId'",
                function ($command, $conn) use ($deferred, $countBought, $roomName, $itemId) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getError());
                        return ;
                    }
                    $countBuying = (int)reset($command->resultRows[0]);
                    if ($countBuying == $countBought) {
                        $deferred->resolve(null);
                    } else {
                        $deferred->reject(sprintf("%s %s %s is already bought", $roomName, $itemId, $countBuying));
                    }
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $totalMilliIsu = gmp_init(0, 10);
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $reqTime, &$addings, &$totalMilliIsu) {
            $db->query(
                "SELECT isu FROM adding WHERE room_name = '$roomName' AND time <= $reqTime",
                function ($command, $conn) use ($deferred, &$addings, &$totalMilliIsu) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getError());
                        return ;
                    }
                    foreach ($command->resultRows as $row) {
                        $a = new Adding;
                        $a->hydrate($row);
                        $totalMilliIsu = gmp_add($totalMilliIsu, gmp_mul($a->isu, "1000"));
                        $deferred->resolve(null);
                    }
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    /** @var Buying[] */
    $buyings = [];
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, &$buyings) {
            $db->query(
                "SELECT item_id, ordinal, time FROM buying WHERE room_name = '$roomName'",
                function ($command, $conn) use ($deferred, &$buyings) {
                    foreach ($command->resultRows as $row) {
                        $obj = new Buying;
                        $obj->hydrate($row);
                        $buyings[] = $obj;
                    }
                    $deferred->resolve(null);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $itemId, &$buyings, &$totalMilliIsu, $reqTime) {
            $retrieveMItem = function ($conn, $itemId) use ($db) {
                $deferred = new React\Promise\Deferred();
                $db->query(
                    "SELECT * FROM m_item WHERE item_id = $itemId",
                    function ($command, $conn) use ($deferred) {
                        if ($command->hasError()) {
                            $deferred->reject($command->getError());
                        } else {
                            $mItem = new MItem;
                            $mItem->hydrate($command->resultRows[0]);
                            $deferred->resolve($mItem);
                        }
                    }
                );
                return $deferred->promise();
            };
            $promises = [];
            foreach ($buyings as $b) {
                $promises[] = $retrieveMItem($db, $b->itemId)->then(function ($mItem) use ($b, &$totalMilliIsu, $reqTime) {
                     $cost = $mItem->getPrice($b->ordinal, $reqTime - $b->time);
                     $totalMilliIsu = gmp_sub($totalMilliIsu, $cost);
                    if ($b->time <= $reqTime) {
                        $gain = gmp_mul($mItem->getPower($b->ordinal), $reqTime - $b->time);
                        $totalMilliIsu = gmp_add($totalMilliIsu, $gain);
                    }
                });
            }
            React\Promise\all($promises)->then(
                function () use ($deferred) {
                    $deferred->resolve(null);
                },
                function ($err) use ($deferred) {
                    $deferred->reject($err);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $itemId, $countBought, &$totalMilliIsu) {
            $db->query(
                "SELECT * FROM m_item WHERE item_id = $itemId",
                function ($command, $conn) use ($deferred, $countBought, &$totalMilliIsu) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getError());
                        return ;
                    }
                    $mItem = new MItem;
                    $mItem->hydrate($command->resultRows[0]);
                    $need = gmp_mul($mItem->getPrice($countBought + 1), "1000");
                    if ($need <= $totalMilliIsu) {
                        $deferred->resolve(null);
                    } else {
                        $deferred->reject("not enough");
                    }
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $itemId, $countBought, $reqTime) {
            $currentBought = $countBought + 1;
            $db->query(
                "INSERT INTO buying(room_name, item_id, ordinal, time) VALUES('$roomName', $itemId, $currentBought, $reqTime)",
                function ($command, $conn) use ($deferred, $countBought) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getError());
                    } else {
                        $deferred->resolve(null);
                    }
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, $reqTime) {
            $db->query(
                "COMMIT",
                function ($command, $conn) use ($deferred) {
                    $deferred->resolve(new GameResponse('', true));
                    $conn->close();
                }
            );
        },
        function ($err) use ($deferred, $db) {
            $db->query("ROLLBACK", function ($_, $conn) use ($deferred, $err) {
                $conn->close();
                getLogger()->error(__METHOD__. "(" . __LINE__ . "): " . $err);
                $deferred->reject(new GameResponse('', false));
            });
        }
    );

    return $deferred->promise();
}

/**
 * @param string $roomName
 * @return React\Promise\Promise
 */
function getStatus(string $roomName)
{
    $db = getDbConnection();

    $deferred = new React\Promise\Deferred();
    $db->query(
        "BEGIN",
        function ($command, $conn) use ($deferred) {
            if ($command->hasError()) {
                $deferred->reject($command->getError());
            } else {
                $deferred->resolve(null);
            }
        }
    );

    $currentTime = 0;
    $promise = updateRoomTime($db, $deferred->promise(), $roomName, 0);
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($time) use (&$currentTime, $deferred) {
            $currentTime = $time;
            $deferred->resolve(null);
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    /** @var MItems[] */
    $mItems = [];
    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred, &$mItems) {
            $db->query(
                "SELECT * FROM m_item",
                function ($command, $conn) use ($deferred, &$mItems) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getError());
                        return ;
                    }
                    foreach ($command->resultRows as $row) {
                        $obj = new MItem;
                        $obj->hydrate($row);
                        $mItems[$obj->itemId] = $obj;
                    }
                    $deferred->resolve(null);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $deferred = new React\Promise\Deferred();
    /** @var Adding[] */
    $addings = [];
    $promise->then(
        function ($t) use ($db, $deferred, $roomName, &$addings) {
            $currentTime = $t;
            $db->query(
                "SELECT time, isu FROM adding WHERE room_name = '$roomName'",
                function ($command, $conn) use ($deferred, $roomName, &$addings) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getError());
                        return ;
                    }
                    foreach ($command->resultRows as $row) {
                        $obj = new Adding;
                        $obj->hydrate($row);
                        $addings[] = $obj;
                    }
                    $deferred->resolve(null);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    /** @var Buying[] */
    $buyings = [];
    $promise->then(
        function ($x) use ($db, $deferred, $roomName, &$buyings) {
            $db->query(
                "SELECT item_id, ordinal, time FROM buying WHERE room_name = '$roomName'",
                function ($command, $conn) use ($deferred, $roomName, &$buyings) {
                    if ($command->hasError()) {
                        $deferred->reject($command->getError());
                        return ;
                    }
                    foreach ($command->resultRows as $row) {
                        $obj = new Buying;
                        $obj->hydrate($row);
                        $buyings[] = $obj;
                    }
                    $deferred->resolve(null);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($db, $deferred) {
            $db->query(
                "COMMIT",
                function ($command, $conn) use ($deferred) {
                    if ($command->hasError()) {
                        $deferred->resolve($command->getError());
                    } else {
                        $deferred->resolve(null);
                    }
                    $conn->close();
                }
            );
        },
        function ($err) use ($db, $deferred) {
            $db->close();
            $deferred->reject($err);
        }
    );

    $promise = $deferred->promise();
    $deferred = new React\Promise\Deferred();
    $promise->then(
        function ($x) use ($deferred, &$currentTime, &$mItems, &$addings, &$buyings, $db) {
            $status = calcStatus($currentTime, $mItems, $addings, $buyings);
            getCurrentTime()->then(
                function ($finishedTime) use ($status, $deferred) {
                    $status->time = $finishedTime;
                    $deferred->resolve($status);
                }
            );
        },
        function ($err) use ($deferred) {
            $deferred->reject($err);
        }
    );
    return $deferred->promise();
}

function calcStatus(int $currentTime, array $mItems, array $addings, array $buyings): GameStatus
{
    // 1ミリ秒に生産できる椅子の単位をミリ椅子とする
    $totalMilliIsu = gmp_init(0, 10);
    $totalPower    = gmp_init(0, 10);

    $itemPower    = []; // ItemId => Power
    $itemPrice    = []; // ItemId => Price
    $itemOnSale   = []; // ItemId => OnSale
    $itemBuilt    = []; // ItemId => BuiltCount
    $itemBought   = []; // ItemId => CountBought
    $itemBuilding = []; // ItemId => Buildings
    $itemPower0   = []; // ItemId => currentTime における Power
    $itemBuilt0   = []; // ItemId => currentTime における BuiltCount

    $addingAt = []; // Time => currentTime より先の Adding
    $buyingAt = []; // Time => currentTime より先の Buying

    foreach ($mItems as $itemId => $_) {
        $itemPower[$itemId] = gmp_init(0, 10);
        $itemBuilding[$itemId] = [];
        $itemBuilt[$itemId] = 0;
        $itemBought[$itemId] = 0;
    }

    foreach ($addings as $a) {
        // adding は adding.time に isu を増加させる
        if ($a->time <= $currentTime) {
            $totalMilliIsu = gmp_add($totalMilliIsu, gmp_mul(gmp_init($a->isu), "1000"));
        } else {
            $addingAt[$a->time] = $a;
        }
    }

    foreach ($buyings as $b) {
        // buying は 即座に isu を消費し buying.time からアイテムの効果を発揮する
        ++$itemBought[$b->itemId];
        $m = $mItems[$b->itemId];
        $totalMilliIsu = gmp_sub($totalMilliIsu, gmp_mul($m->getPrice($b->ordinal), "1000"));

        if ($b->time <= $currentTime) {
            ++$itemBuilt[$b->itemId];
            $power = $m->getPower($itemBought[$b->itemId]);
            $totalMilliIsu = gmp_add($totalMilliIsu, gmp_mul($power, $currentTime - $b->time));
            $totalPower = gmp_add($totalPower, $power);
            $itemPower[$b->itemId] = gmp_add($itemPower[$b->itemId], $power);
        } else {
            $buyingAt[$b->time][] = $b;
        }
    }

    foreach ($mItems as $m) {
        $itemPower0[$m->itemId] = big2exp($itemPower[$m->itemId]);
        $itemBuilt0[$m->itemId] = $itemBuilt[$m->itemId];
        $price = $m->getPrice($itemBought[$m->itemId] + 1);
        $itemPrice[$m->itemId] = $price;
        if ($totalMilliIsu >= gmp_mul($price, "1000")) {
            $itemOnSale[$m->itemId] = 0; // 0 は 時刻 currentTime で購入可能であることを表す
        }
    }

    /** @var Schedule[] */
    $schedule = [];
    $schedule[] = new Schedule($currentTime, big2exp($totalMilliIsu), big2exp($totalPower));

    // currentTime から 1000 ミリ秒先までシミュレーションする
    for ($t = $currentTime + 1; $t <= $currentTime + 1000; ++$t) {
        $totalMilliIsu = gmp_add($totalMilliIsu, $totalPower);
        $updated = false;

        // 時刻 t で発生する adding を計算する
        if (isset($addingAt[$t])) {
            $a = $addingAt[$t];
            $updated = true;
            $totalMilliIsu = gmp_add($totalMilliIsu, gmp_mul(str2big($a->isu), "1000"));
        }

        // 時刻 t で発生する buying を計算する
        if (isset($buyingAt[$t])) {
            $updated = true;
            $updatedId = [];
            foreach ($buyingAt[$t] as $b) {
                $m = $mItems[$b->itemId];
                $updatedId[$b->itemId] = $b->itemId;
                ++$itemBuilt[$b->itemId];
                $power = $m->getPower($b->ordinal);
                $itemPower[$b->itemId] = gmp_add($itemPower[$b->itemId], $power);
                $totalPower = gmp_add($totalPower, $power);
            }
            foreach ($updatedId as $id) {
                $itemBuilding[$id][] = new Building(
                    $t,
                    $itemBuilt[$id],
                    big2exp($itemPower[$id])
                );
            }
        }

        if ($updated) {
            $schedule[] = new Schedule(
                $t,
                big2exp($totalMilliIsu),
                big2exp($totalPower)
            );
        }

        // 時刻 t で購入可能になったアイテムを記録する
        foreach ($mItems as $itemId => $_) {
            if (isset($itemOnSale[$itemId])) {
                continue;
            }
            if ($totalMilliIsu >= gmp_mul($itemPrice[$itemId], "1000")) {
                $itemOnSale[$itemId] = $t;
            }
        }
    }

    /** @var Adding[] */
    $gsAdding = array_values($addingAt);

    /** @var Item[] */
    $gsItems = [];
    foreach ($mItems as $itemId => $_) {
        $gsItems[] = new Item(
            $itemId,
            $itemBought[$itemId],
            $itemBuilt0[$itemId],
            big2exp($itemPrice[$itemId]),
            $itemPower0[$itemId],
            $itemBuilding[$itemId]
        );
    }

    /** @var OnSale[] */
    $gsOnSale = [];
    foreach ($itemOnSale as $itemId => $t) {
        $gsOnSale[] = new OnSale($itemId, $t);
    }

    return new GameStatus($currentTime, $gsAdding, $schedule, $gsItems, $gsOnSale);
}
