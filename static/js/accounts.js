
app.controller('AccountViewController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.accounts = [];
  $scope.error = null;

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'accounts'){
      $scope.loading = true;
      $http({
        method: 'GET',
        url: '/web/v1/accounts'
      }).then(function successCallback(response) {
        $scope.loading = false;
        $scope.accounts = response.data;
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }
  });
}]);
