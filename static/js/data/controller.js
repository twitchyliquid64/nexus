
app.controller('DatastoreController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.datastores = [];
  $scope.error = null;

  $scope.update = function(){
    $scope.loading = true;
    $http({
      method: 'GET',
      url: '/web/v1/data/list'
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.datastores = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.create = function(){
    $rootScope.$broadcast('create-datastore', {cb: function(ds, cols){
      $http({
        method: 'POST',
        url: '/web/v1/data/new',
        data: {Datastore: ds, Cols: cols},
      }).then(function successCallback(response) {
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }});
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'data'){
      $scope.update();
    }
  });
}]);
