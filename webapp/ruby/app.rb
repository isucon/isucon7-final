require 'erb'
require 'sinatra/base'
require './game'

class App < Sinatra::Base
  use Game

  configure do
    set :public_folder, File.expand_path('../../public', __FILE__)
  end

  configure :development do
    require 'sinatra/reloader'
    register Sinatra::Reloader
  end

  get '/initialize' do
    Game.initialize!
    204
  end

  get '/room/' do
    content_type :json
    { 'host' => '', 'path' => '/ws' }.to_json
  end

  get '/room/:room_name' do
    room_name = ERB::Util.url_encode(params[:room_name])
    path = "/ws/#{room_name}"

    content_type :json
    { 'host' => '', 'path' => path }.to_json
  end

  get '/' do
    send_file File.expand_path('index.html', settings.public_folder)
  end
end
