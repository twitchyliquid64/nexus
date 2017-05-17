
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

  $scope.explore = function(ds){
    $rootScope.$broadcast('data-explore', {ds: ds});
    $scope.changePage('data-explorer');
  }

  $scope.insert = function(ds){
    $rootScope.$broadcast('data-insert', {ds: ds});
    $scope.changePage('data-inserter');
  }

  $scope.delete = function(uid){
    $rootScope.$broadcast('check-confirmation', {
      title: 'Confirm Deletion',
      content: 'Are you sure you want to delete datastore \'' + uid + '\'?',
      actions: [
        {text: 'No'},
        {text: 'Yes', onAction: function(){
          $scope.loading = true;
          $http({
            method: 'POST',
            url: '/web/v1/data/delete',
            data: [uid],
          }).then(function successCallback(response) {
            $scope.update();
          }, function errorCallback(response) {
            $scope.loading = false;
            $scope.error = response;
          });
        }},
      ]
    });
  }

  $scope.edit = function(ds){
    console.log(ds);
    $rootScope.$broadcast('edit-datastore', {ds: ds, cb: function(ds, cols){
      ds.Cols = cols;
      $http({
        method: 'POST',
        url: '/web/v1/data/edit',
        data: ds,
      }).then(function successCallback(response) {
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }});
  }

  $scope.create = function(){
    $rootScope.$broadcast('create-datastore', {cb: function(ds, cols){
      ds.Cols = cols;
      $http({
        method: 'POST',
        url: '/web/v1/data/new',
        data: ds,
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
