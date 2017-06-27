app.controller('IntegrationRunExplorer', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.runnable = null;
  $scope.runs = [];
  $scope.error = null;

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = null;
     $http({
       method: 'POST',
       url: '/web/v1/integrations/log/runs',
       data: [$scope.runnable.UID],
     }).then(function successCallback(response) {
       $scope.loading = false;
       $scope.runs = response.data;
       console.log($scope.runs);
     }, function errorCallback(response) {
       $scope.loading = false;
       $scope.error = response;
     });
  }

  $scope.filtersChanged = function(run, filters, limit, offset){
    $scope.filters = filters;
    $scope.limit = limit;
    $scope.offset = offset;
    console.log("New constraints: ", run, filters, limit, offset);
  }

  $rootScope.$on('integration-run-explorer', function(event, args) {
    $scope.runnable = args.runnable;
    $scope.runs = [];
  });

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'integration-run-explorer'){
      $scope.update();
    }
  });

}]);
