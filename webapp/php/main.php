<?php

declare(strict_types=1);

use GuzzleHttp\Psr7 as gPsr;
use GuzzleHttp\Psr7\Response;
use Psr\Http\Message\RequestInterface;
use Ratchet\ConnectionInterface;
use Ratchet\Http\HttpServer;
use Ratchet\Http\HttpServerInterface;
use Ratchet\MessageComponentInterface;
use Ratchet\Server\IoServer;
use Ratchet\WebSocket\WsServer;

require_once __DIR__ . '/vendor/autoload.php';
require_once __DIR__ . '/game.php';

class Throttle
{
    /** @var int */
    private $concurrentLimit = 100;

    /** @var array */
    private $waitingQueue = [];

    /** @var int */
    private $concurrency = 0;

    public function __construct(int $concurrentLimit)
    {
        $this->$concurrentLimit = $concurrentLimit;
    }

    /**
     * @return React\Promise\Deferred
     */
    public function start()
    {
        $deferred = new React\Promise\Deferred();
        if ($this->concurrency < $this->concurrentLimit) {
            ++$this->concurrency;
            $deferred->resolve(null);
        } else {
            array_push($this->waitingQueue, $deferred);
        }
        return $deferred->promise();
    }

    public function finish()
    {
        if (!empty($this->waitingQueue)) {
            $deferred = array_shift($this->waitingQueue);
            $deferred->resolve();
        } else {
            --$this->concurrency;
        }
    }
}

class RoomController implements HttpServerInterface
{
    public function onOpen(ConnectionInterface $conn, Psr\Http\Message\RequestInterface $request = null)
    {
        $response = new Response(200, [
            'Content-Type' => 'application/json',
        ]);

        $param = [];
        $uri = (string)$request->getUri();
        parse_str(parse_url($uri)['query'] ?? '', $param);

        $data = [
            'host' => '',
            'path' => '/ws/' . ($param['room_name'] ?? ''),
        ];
        getLogger()->info(json_encode($data));
        $response->getBody()->write(json_encode($data));
        $conn->send(gPsr\str($response));
        $conn->close();
    }

    public function onMessage(ConnectionInterface $from, $msg)
    {
    }

    public function onClose(ConnectionInterface $conn)
    {
        $conn->close();
    }

    public function onError(ConnectionInterface $conn, \Exception $e)
    {
        getLogger()->error($e->getMessage());
        $conn->close();
    }
}

class InitializeController implements HttpServerInterface
{
    public function onOpen(ConnectionInterface $conn, Psr\Http\Message\RequestInterface $request = null)
    {
        $db = getDbConnection();

        $deferred = new React\Promise\Deferred();
        $promise = $deferred->promise();
        $promise->then($db->query("TRUNCATE TABLE adding", function ($command, $conn) use ($deferred) {
            $deferred->resolve();
        }));

        $deferred = new React\Promise\Deferred();
        $promise = $deferred->promise();
        $db->query("TRUNCATE TABLE adding", function ($command, $conn) use ($deferred) {
            $deferred->resolve();
        });

        $deferred = new React\Promise\Deferred();
        $promise = $deferred->promise();
        $db->query("TRUNCATE TABLE buying", function ($command, $conn) use ($deferred) {
            $deferred->resolve();
        });

        $deferred = new React\Promise\Deferred();
        $promise = $deferred->promise();
        $db->query("TRUNCATE TABLE room_time", function ($command, $conn) use ($deferred) {
            $deferred->resolve();
        });

        $promise->then(function ($_) use ($conn, $db) {
            $response = new Response(204, []);
            $conn->send(gPsr\str($response));
            $conn->close();
            $db->close();
        });
    }

    public function onMessage(ConnectionInterface $from, $msg)
    {
    }

    public function onClose(ConnectionInterface $conn)
    {
        $conn->close();
    }

    public function onError(ConnectionInterface $conn, \Exception $e)
    {
        getLogger()->error($e->getMessage());
        $conn->close();
    }
}

class GameController implements MessageComponentInterface
{
    private function getThrottle()
    {
        static $throttle = null;
        if (is_null($throttle)) {
            // 同時に処理する手続きに上限を設けないと、DBとのコネクション数が限界を迎えるので
            // resolveされるdeferredを制限する。
            $throttle = new Throttle(80);
        }
        return $throttle;
    }

    private function parseRoomName(RequestInterface $request)
    {
        $parsed = parse_url((string)$request->getUri());
        $output = [];
        parse_str($parsed['query'] ?? '', $output);
        return $output['room_name'] ?? '';
    }

    public function onOpen(ConnectionInterface $conn)
    {
        global $loop;
        $roomName = $this->parseRoomName($conn->httpRequest);
        getLogger()->info(__LINE__ . ": " . "serveGameConn(" . $roomName . ")");
        $conn->roomName = $roomName;
        $this->resumeTimer($conn);
    }

    private function stopTimer(ConnectionInterface $conn)
    {
        global $loop;
        if (isset($conn->timer) && $loop->isTimerActive($conn->timer)) {
            $loop->cancelTimer($conn->timer);
        }
        unset($conn->timer);
    }

    private function resumeTimer(ConnectionInterface $conn)
    {
        global $loop;
        if (isset($conn->timer) && $loop->isTimerActive($conn->timer)) {
            return ;
        }
        if (isset($conn->_closed)) {
            return ;
        }

        $onSucc = function ($_) use ($conn) {
            getStatus($conn->roomName)->then(
                function (GameStatus $status) use ($conn) {
                    $conn->send(json_encode($status));
                    $this->getThrottle()->finish();
                },
                function ($err) {
                    getLogger()->error($err);
                    $this->getThrottle()->finish();
                }
            );
        };
        $onFail = function ($err) {
            getLogger()->error($err);
            $this->getThrottle()->finish();
        };
        $conn->timer = $loop->addPeriodicTimer(
            0.5,
            function () use ($onSucc, $onFail) {
                $this->getThrottle()->start()->then($onSucc, $onFail);
            }
        );
    }

    public function onMessage(ConnectionInterface $conn, $msg)
    {
        $this->stopTimer($conn);

        getLogger()->info(__LINE__ . ": " . substr($msg, 0, -1) . ", " . $conn->roomName);
        $assoc = json_decode($msg, true);
        $request = new GameRequest();
        $request->hydrate($assoc);

        $fn = function (GameResponse $response) use ($conn, $request) {
            $response->requestId = $request->requestId;
            if ($response->isSuccess) {
                getStatus($conn->roomName)->then(
                    function (GameStatus $status) use ($conn, $response) {
                        // GameResponse を返却する前に 反映済みの GameStatus を返す
                        $conn->send(json_encode($status));
                        $conn->send(json_encode($response));
                        $this->getThrottle()->finish();
                        $this->resumeTimer($conn);
                    },
                    function ($err) {
                        $this->getThrottle()->finish();
                        getLogger()->error($err);
                        $this->resumeTimer($conn);
                    }
                );
            } else {
                $conn->send(json_encode($response));
                $this->getThrottle()->finish();
                $this->resumeTimer($conn);
            }
        };

        $this->getThrottle()->start()->then(
            function ($_) use ($request, $conn, $fn) {
                switch ($request->action) {
                    case 'addIsu':
                        addIsu($conn->roomName, gmp_init($request->isu), $request->time)->then($fn, $fn);
                        break;
                    case 'buyItem':
                        buyItem($conn->roomName, $request->itemId, $request->countBought, $request->time)->then($fn, $fn);
                        break;
                    default:
                        getLogger()->error("Invalid Action");
                        $this->getThrottle()->finish();
                        break;
                }
            }
        );
    }

    public function onClose(ConnectionInterface $conn)
    {
        getLogger()->info("connection(" . $conn->resourceId . ") is closed");
        $this->stopTimer($conn);
        $conn->_closed = true;
        $conn->close();
    }

    public function onError(ConnectionInterface $conn, \Exception $e)
    {
        getLogger()->error($e->getMessage());
        $this->stopTimer($conn);
        $conn->_closed = true;
        $conn->close();
    }
}

/**
 * Ratchet\AppはWsServerを受け取ると勝手にenableKeepAliveを呼び出してしまうので封じる
 */
class NoKeepAliveWsServer extends WsServer
{
    public function enableKeepAlive(React\EventLoop\LoopInterface $loop, $interval = 30)
    {
    }
}

class StaticFileController implements HttpServerInterface
{
    private $filepath;

    public function __construct(string $filepath)
    {
        $this->filepath = $filepath;
    }

    public function onOpen(ConnectionInterface $conn, Psr\Http\Message\RequestInterface $request = null)
    {
        getLogger()->info(__DIR__ . '/../public/' . $this->filepath);
        $response = new Response(200, []);
        $response->getBody()->write(file_get_contents(__DIR__ . '/../public/' . $this->filepath));
        $conn->send(gPsr\str($response));
        $conn->close();
    }

    public function onMessage(ConnectionInterface $from, $msg)
    {
    }

    public function onClose(ConnectionInterface $conn)
    {
        $conn->close();
    }

    public function onError(ConnectionInterface $conn, \Exception $e)
    {
        $conn->close();
    }
};

$room = new RoomController;
$initialize = new InitializeController;
$ws = new NoKeepAliveWsServer(new GameController);

$app = new Ratchet\App('127.0.0.1', 5000, '127.0.0.1', $loop);
$app->route('/initialize', $initialize, ['*']);
$app->route('/ws', $ws, ['*']);
$app->route('/ws/', $ws, ['*']);
$app->route('/ws/{room_name}', $ws, ['*']);
$app->route('/room', $room, ['*']);
$app->route('/room/', $room, ['*']);
$app->route('/room/{room_name}', $room, ['*']);

$pathes = scandir(__DIR__ . '/../public/');
foreach ($pathes as $k => $path) {
    if ($path == '.') {
        continue;
    }
    if ($path == '..') {
        continue;
    }
    $app->route($path, new StaticFileController($path), ['*']);
}
$pathes = scandir(__DIR__ . '/../public/images/');
foreach ($pathes as $k => $path) {
    if ($path == '.') {
        continue;
    }
    if ($path == '..') {
        continue;
    }
    $app->route('images/' . $path, new StaticFileController('images/' . $path), ['*']);
}
$app->route('/', new StaticFileController('index.html'), ['*']);

// 黙って死ぬことがあるのでログに残す
register_shutdown_function(function () { getLogger()->error("DIE"); });
$app->run();
