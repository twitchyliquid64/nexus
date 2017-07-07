
app.controller('MessengerController', ["$scope", "$rootScope", "$http", "$interval",  function ($scope, $rootScope, $http, $interval) {
  $scope.loading = false;
  $scope.baseData = [];
  $scope.error = null;
  $scope.selected = 0;
  $scope.currentConvoTitle = '';
  $scope.currentConvoMessages = [];
  $scope.msg = '';

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = undefined;
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
    $scope.currentConvoMessages = [];
    $scope.loadCurrentConvo(convo);
  }

  $scope.send = function(){
    if($scope.msg){
      $scope.error = undefined;
      $scope.loading = true;
      var a = $scope.msg;
      $scope.msg = '';
      $http({
        method: 'POST',
        data: {cid: $scope.selected, msg: a},
        url: '/web/v1/messenger/send',
      }).then(function successCallback(response) {
        $scope.loading = false;
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }
  }

  $scope.loadCurrentConvo = function(convo){
    $scope.loading = true;
    $scope.error = undefined;
    $http({
      method: 'GET',
      url: '/web/v1/messenger/messages?cid=' + convo.UID
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.currentConvoMessages = response.data;
      $scope.$$postDigest(function(){
        var objDiv = document.getElementById("messages-container");
        objDiv.scrollTop = objDiv.scrollHeight;
      });
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

  $interval(function(){
    if ($scope.selected){
      $scope.loadCurrentConvo({UID: $scope.selected});
    }
  }, 5500);
}]);
