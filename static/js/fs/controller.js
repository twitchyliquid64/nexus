
app.controller('FSController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.files = [];
  $scope.error = null;
  $scope.path = "/";

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'POST',
      url: '/web/v1/fs/list',
      data: {path: $scope.path},
    }).then(function successCallback(response) {
      $scope.loading = false;
      if (response.data && response.data.success == false){
        $scope.error = response.data;
        return
      }
      $scope.files = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.back = function(){
    var spl = $scope.path.split('/');
    $scope.path = $scope.path.substring(0, $scope.path.length - spl[spl.length-1].length-1);
    if (!$scope.path || $scope.path == '') {
      $scope.path = '/';
    }
    $scope.update();
  }

  $scope.name = function(file){
    var spl = file.Name.split('/');
    return spl[spl.length-1];
  }
  $scope.icon = function(file){
    switch (file.ItemKind){
      case 1://root
        return 'dns';
      case 2://file
        return 'insert_drive_file';
      case 3://folder
        return 'folder';
    }
    return 'help_outline'
  }

  $scope.date = function(file){
    if (file.Modified == "0001-01-01T00:00:00Z")return "";
    return moment(file.Modified).format("dddd, MMMM Do YYYY, h:mm:ss a");
  }

  $scope.nav = function(f){
    if (f.ItemKind == 2) {//file

    } else if ($scope.path == '/'){
      $scope.path = f.Name;
    } else {
      console.log($scope.path.split('/'), f);
      $scope.path = '/' + $scope.path.split('/')[1] + f.Name;
    }
    $scope.update();
  }

  $scope.newFile = function(){
    var newName = prompt("New file name (including extension):");
    if (newName){
      $scope.loading = true;
      $scope.error = null;
      $http({
        method: 'POST',
        url: '/web/v1/fs/save',
        data: {path: $scope.path + '/' + newName},
      }).then(function successCallback(response) {
        $scope.loading = false;
        if (response.data && response.data.success == false){
          $scope.error = response.data;
          return
        }
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }
  }

  $scope.newFolder = function(){
    var newName = prompt("New folder name:");
    if (newName){
      $scope.loading = true;
      $scope.error = null;
      $http({
        method: 'POST',
        url: '/web/v1/fs/newFolder',
        data: {path: $scope.path + '/' + newName},
      }).then(function successCallback(response) {
        $scope.loading = false;
        if (response.data && response.data.success == false){
          $scope.error = response.data;
          return
        }
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }
  }

  $scope.delete = function(file){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'POST',
      url: '/web/v1/fs/delete',
      data: {path: '/' + $scope.path.split('/')[1] + file.Name},
    }).then(function successCallback(response) {
      $scope.loading = false;
      if (response.data && response.data.success == false){
        $scope.error = response.data;
        return
      }
      $scope.update();
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'files'){
      $scope.update();
    } else {
      $scope.path = "/";
      $scope.files = [];
    }
  });
}]);
