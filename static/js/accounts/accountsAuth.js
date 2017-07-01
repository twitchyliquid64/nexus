
app.controller('AccountsAuthController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.account = null;
  $scope.auths = [];
  $scope.error = null;

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'POST',
      url: '/web/v1/account/auths',
      data: $scope.account.UID,
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.auths = response.data;
      $scope.$$postDigest(function(){
        $('.collapsible').collapsible();
      })
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $rootScope.$on('accounts-auth-setup', function(event, args) {
    $scope.account = args.user;
  });

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'accounts-auth'){
      $scope.update();
    }
  });
}]);
