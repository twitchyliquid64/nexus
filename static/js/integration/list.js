app.controller('IntegrationsController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.integrations = [];
  $scope.error = null;

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'GET',
      url: '/web/v1/integrations/mine'
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.integrations = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.delete = function(obj){
    $rootScope.$broadcast('check-confirmation', {
      title: 'Confirm Deletion',
      content: 'Are you sure you want to delete the integration \'' + obj.Name + '\' (' + obj.UID + ')?',
      actions: [
        {text: 'No'},
        {text: 'Yes', onAction: function(){
          $scope.loading = true;
          $scope.error = null;
          $http({
            method: 'POST',
            url: '/web/v1/integrations/delete/runnable',
            data: [obj.UID],
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

  $scope.createIntegration = function(){
    $rootScope.$broadcast('create-integration',{cb: function(newIntegration, triggers){
      console.log("New integration", newIntegration);
      newIntegration.Triggers = triggers;

      $scope.loading = true;
      $scope.error = null;
      $http({
        method: 'POST',
        url: '/web/v1/integrations/create/runnable',
        data: newIntegration,
      }).then(function successCallback(response) {
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }});
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'integrations'){
      $scope.update();
    }
  });

}]);
