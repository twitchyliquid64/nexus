
app.controller('AccountsAuthController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.account = null;
  $scope.auths = [];
  $scope.error = null;
  $scope.inputType = "password";
  $scope.imgdata = null;

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

  $scope.icon = function(auth){
    switch (auth.Kind){
      case 0:
        return 'vpn_key'; //OTP
    }
    return 'security';
  }

  $scope.cName = function(c){
    if (c == 0)return 'Required';
    return 'Optional';
  }
  $scope.kName = function(k){
    if (k == 0)return 'OTP';
    return 'Password';
  }

  $scope.genOTP = function(){
    $scope.loading = true;
    $scope.error = null;
    if ($scope.new.Val2 == "") {
      $scope.new.Val2 = "OTP";
    }
    $http({
      method: 'POST',
      url: '/web/v1/genotp',
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      data: 'account='+$scope.new.Val2,
    }).then(function successCallback(response) {
      $scope.loading = false;
      console.log(response.data);
      $scope.new.Val1 = response.data.Key;
      $scope.new.Kind = 0;
      $scope.inputType = 'text';
      $scope.imgdata = response.data.Img;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.newAuth = function(){
    $scope.loading = true;
    $scope.error = null;
    $scope.new.UserID = $scope.account.UID;
    $http({
      method: 'POST',
      url: '/web/v1/account/addauth',
      data: $scope.new,
    }).then(function successCallback(response) {
      $scope.update();
      $scope.resetNew();
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.deleteAuth = function(auth){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'POST',
      url: '/web/v1/account/delauth',
      data: [auth.UID],
    }).then(function successCallback(response) {
      $scope.update();
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.resetNew = function(){
    $scope.inputType = 'password';
    $scope.imgdata = null;
    $scope.new = {
      Kind: 1,
      Class: 1,
      Val1: '',
      Val2: '',
    };
  }

  $rootScope.$on('accounts-auth-setup', function(event, args) {
    $scope.account = args.user;
    $scope.resetNew();
  });

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'accounts-auth'){
      $scope.update();
    }
  });
}]);
