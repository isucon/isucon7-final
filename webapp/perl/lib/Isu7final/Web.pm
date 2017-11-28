package Isu7final::Web;

use Mojolicious::Lite;
use Mojo::Util qw(url_escape);
use Mojo::mysql;

plugin Config => { file => app->home->path("../../config.pl") };

helper mysql => sub {
    state $mysql = do {
        my %db = (
            host     => $ENV{ISU_DB_HOST} || 'localhost',
            port     => $ENV{ISU_DB_PORT} || 3306,
            username => $ENV{ISU_DB_USER} || 'root',
            password => $ENV{ISU_DB_PASSWORD} ? ":".$ENV{ISU_DB_PASSWORD} : '',
        );

        my $mysql = Mojo::mysql->new("mysql://$db{username}$db{password}\@$db{host}:$db{port}/isudb");

        # http://search.cpan.org/~jhthorsen/Mojo-mysql/README.pod#strict_mode
        $mysql->strict_mode(1); # SET SQL_MODE = CONCAT('ANSI,TRADITIONAL,ONLY_FULL_GROUP_BY,', @@sql_mode)
                                # SET SQL_AUTO_IS_NULL = 0

        $mysql->on(connection => sub {
            my ($mysql, $dbh) = @_;

            $dbh->do("SET NAMES utf8mb4");
        });

        $mysql;
    };
};

plugin 'Isu7final::Game';

get '/initialize' => sub {
    my $c = shift;

    $c->mysql->db->query("TRUNCATE TABLE adding");
    $c->mysql->db->query("TRUNCATE TABLE buying");
    $c->mysql->db->query("TRUNCATE TABLE room_time");

    $c->render(text => '', status => 204);
};

get '/room/:room_name' => { room_name => undef } => sub {
    my $c = shift;

    my $room_name = $c->param('room_name') // '';
    my $path      = "/ws/" . url_escape($room_name);

    $c->render(json => { host => '', path => $path });
};

websocket '/ws/:room_name' => { room_name => undef } => sub {
    my $c = shift;

    my $room_name = $c->param('room_name') // "";

    $c->serve_game_conn($room_name);
};

# serve static file
app->static->paths->[0] = app->home->path('../../../public');

# / => index.html
get '/' => sub {
    my $c = shift;

    $c->reply->static('index.html');
};

1;
