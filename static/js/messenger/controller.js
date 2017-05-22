
app.controller('MessengerController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.baseData = [];
  $scope.error = null;
  $scope.selected = 0;
  $scope.currentConvoTitle = '';
  $scope.currentConvoMessages = [];

  $scope.update = function(){
    $scope.loading = true;
    $http({
      method: 'GET',
      url: '/web/v1/messenger/conversations'
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.baseData = response.data;
      $scope.openConvo($scope.baseData.Conversations[0]);
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.openConvo = function(convo){
    $scope.selected = convo.UID;
    $scope.currentConvoTitle = convo.Name;
    $scope.loadCurrentConvo(convo);
  }

  $scope.loadCurrentConvo = function(convo){
    $scope.loading = true;
    $http({
      method: 'GET',
      url: '/web/v1/messenger/messages?cid=' + convo.UID
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.currentConvoMessages = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.getConvoSecondLine = function(convo){
    for (var i = 0; i < $scope.baseData.Sources.length; i++) {
      if (convo.SourceUID == $scope.baseData.Sources[i].UID) {
        return $scope.baseData.Sources[i].Name;
      }
    }
    return "?";
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'messenger'){
      $scope.update();
    }
  });
}]);
