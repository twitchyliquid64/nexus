app.controller('FileEditorController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.editorObj = null;
  $scope.langTools = null;
  $scope.lastSaved = moment();
  $scope.loading = false;
  $scope.error = null;
  $scope.selectedMode = 'text';

  $scope.path = null;
  $scope.file = null;
  $scope.content = null;

  $scope.save = function(){
    $scope.loading = true;
    $scope.error = null;
    return new Promise(function(resolve, reject) {
      $http({
        method: 'POST',
        url: '/web/v1/fs/save',
        data: {
          path: $scope.path,
          data: $scope.editorObj.getValue(),
        },
      }).then(function successCallback(response) {
        $scope.loading = false;
        $scope.lastSaved = moment();
        resolve();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
        reject(response);
      });
    });
  }

  $scope.loadData = function(){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'GET',
      transformResponse: [function (data) {return data;}],
      url: '/web/v1/fs/download' + $scope.path,
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.content = response.data;
      $scope.editorObj.setValue($scope.content);
      $scope.editorObj.gotoLine(0,0);
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $rootScope.$on('files-editor', function(event, args) {
    function endsWith(str, suffix) {
        return str.indexOf(suffix, str.length - suffix.length) !== -1;
    }
    $scope.file = args.file;
    $scope.path = args.path;
    $scope.lastSaved = moment(args.file.Modified);
    if (endsWith($scope.path, '.html')){
      $scope.selectedMode = 'html';
    }
    if (endsWith($scope.path, '.txt')){
      $scope.selectedMode = 'text';
    }
    if (endsWith($scope.path, '.js')){
      $scope.selectedMode = 'javascript';
    }
    if (endsWith($scope.path, '.json')){
      $scope.selectedMode = 'json';
    }
    $scope.loadData();
  });

  $scope.back = function(){
    $scope.save().then(function(){
      $rootScope.$broadcast('files-navigate', {path: $scope.path.substring(0,$scope.path.lastIndexOf("/"))});
      $scope.changePage('files');
    });
  }

  $scope.modeChange = function(){
    $scope.editorObj.session.setMode("ace/mode/" + $scope.selectedMode);
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'files-editor'){

      if (!$scope.editorObj) {
        $scope.editorObj = ace.edit("fileEditor");
        $scope.editorObj.session.setMode("ace/mode/text");
        $scope.editorObj.setTheme("ace/theme/github");
        $scope.modeChange();
        $scope.editorObj.commands.addCommand({
          name: 'saveFile',
          bindKey: {
            win: 'Ctrl-S',
            mac: 'Command-S',
            sender: 'editor|cli'
          },
          exec: function(env, args, request) {
            $scope.save();
          }
        });
      }
      if ($scope.content){
        $scope.editorObj.setValue($scope.content);
        $scope.editorObj.gotoLine(0,0);
        $scope.modeChange();
      }
      $scope.editorObj.resize();
    }
  });

}]);
