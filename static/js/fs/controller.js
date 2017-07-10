
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

  $scope.name = function(file){
    return file.Name.substring(1);
  }
  $scope.icon = function(file){
    switch (file.ItemKind){
      case 1://root
        return 'dns';
    }
    return 'help_outline'
  }
  $scope.nav = function(f){
    $scope.path = f.Name;
    $scope.update();
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
