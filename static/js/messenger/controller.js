
app.controller('MessengerController', ["$scope", "$rootScope", "$http", "$interval",  function ($scope, $rootScope, $http, $interval) {
  $scope.loading = false;
  $scope.baseData = [];
  $scope.error = null;
  $scope.selected = 0;
  $scope.currentConvoTitle = '';
  $scope.currentConvoMessages = [];
  $scope.msg = '';
  $scope.searchText = '';

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = undefined;
    $http({
      method: 'GET',
      url: '/web/v1/messenger/conversations'
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.baseData = response.data;
      for (var i = 0; i < $scope.baseData.Conversations.length; i++) {
        $scope.baseData.Conversations[i].convo_name = $scope.calcConvoName($scope.baseData.Conversations[i]);
        $scope.baseData.Conversations[i].activityColor = $scope.calcActivityColor($scope.baseData.Conversations[i].NumRecentMessages);
      }
      console.log($scope.baseData);
      $scope.openConvo($scope.baseData.Conversations[0]);
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.calcActivityColor = function(percent){
    var color1 = 'F78400';
    var color2 = '220700';
    var ratio = percent > 11 ? 1 : percent / 11;
    var hex = function(x) {
        x = x.toString(16);
        return (x.length == 1) ? '0' + x : x;
    };

    var r = Math.ceil(parseInt(color1.substring(0,2), 16) * ratio + parseInt(color2.substring(0,2), 16) * (1-ratio));
    var g = Math.ceil(parseInt(color1.substring(2,4), 16) * ratio + parseInt(color2.substring(2,4), 16) * (1-ratio));
    var b = Math.ceil(parseInt(color1.substring(4,6), 16) * ratio + parseInt(color2.substring(4,6), 16) * (1-ratio));

    console.log(ratio, r, g, b);
    return '#' + hex(r) + hex(g) + hex(b);
  }

  $scope.openConvo = function(convo){
    $scope.selected = convo.UID;
    $scope.currentConvoTitle = convo.Name;
    $scope.currentConvoMessages = [];
    $scope.lastMsgTime = null;
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

  $scope.doScroll = function(){
    $scope.$$postDigest(function(){
      var objDiv = document.getElementById("messages-container");
      objDiv.scrollTop = objDiv.scrollHeight;
    });
  }

  $scope.loadCurrentConvo = function(convo){
    $scope.loading = true;
    $scope.error = undefined;
    $http({
      method: 'GET',
      url: '/web/v1/messenger/messages?cid=' + convo.UID
    }).then(function successCallback(response) {
      $scope.loading = false;
      if(response.data && response.data.length && $scope.lastMsgTime){
        if (response.data[0].CreatedAt == $scope.lastMsgTime)
          return;
      }
      $scope.lastMsgTime = response.data ? response.data[0].CreatedAt : null;
      $scope.currentConvoMessages = response.data ? response.data.reverse() : [];
      console.log('updating model');
      $scope.doScroll();
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.calcConvoName = function(convo){
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
    } else {
      $scope.baseData = [];
      $scope.currentConvoMessages = [];
      $scope.lastMsgTime = null;
    }
  });

  $interval(function(){
    if ($scope.selected){
      $scope.loadCurrentConvo({UID: $scope.selected});
    }
  }, 5500);
}]);
