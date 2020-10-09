<?php
namespace Deployer;

require 'recipe/common.php';

// Project name
set('application', 'pisc');

// Project repository
set('repository', 'git@github.com:bobobobob/simleimageserve.git');

// [Optional] Allocate tty for git clone. Default value is false.
set('git_tty', true); 

// Shared files/dirs between deploys 
set('shared_files', []);
set('shared_dirs', []);

// Writable dirs by web server 
set('writable_dirs', []);
set('allow_anonymous_stats', false);
set('keep_releases', 1);
set('http_user', 'www-data');
set('writable_mode', 'chmod');

// Hosts

host('194.67.112.124')
->stage('production')
->user('root')
// ->become('www-data')
// ->configFile('~/.ssh/config')
->identityFile('~/.ssh/id_rsa')
->forwardAgent(true)
->set('deploy_path', '~/{{application}}');    
    

// Tasks
task('srv_stop', 'service imageserver stop')->setPrivate();

task('srv_start', 'service imageserver start')->setPrivate();

task('upload', function () {
    upload('imagesamenu', '/srv/imageserver');
    upload('config.xml', '/srv');
});

desc('Deploy your project');
task('deploy', [
    'deploy:info',
    'deploy:prepare',
    'deploy:lock',
    'deploy:release',
    'srv_stop',
    'upload',
    'srv_start',
    'deploy:unlock',
    'cleanup',
    'success'
]);

// [Optional] If deploy fails automatically unlock.
after('deploy:failed', 'deploy:unlock');
