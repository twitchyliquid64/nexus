
app.controller('AccountViewController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.accounts = [];
  $scope.error = null;

  $scope.edit = function(user){
    console.log("edit", user);
    $rootScope.$broadcast('edit-account',{account: user, cb: function(newUser){
      console.log("Post-edit user", newUser);

      $scope.loading = true;
      $http({
        method: 'POST',
        url: '/web/v1/account/edit',
        data: newUser,
      }).then(function successCallback(response) {
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }});
  }

  $scope.changeAuth = function(user){
    var pass = prompt("Enter the new password for '" + user.DisplayName + "'");
    if (pass) {
      $scope.loading = true;
      $http({
        method: 'POST',
        url: '/web/v1/account/setbasicpass',
        data: {UID: user.UID, Pass: pass},
      }).then(function successCallback(response) {
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }
  }

  $scope.delete = function(uid){
    $rootScope.$broadcast('check-confirmation', {
      title: 'Confirm Deletion',
      content: 'Are you sure you want to delete user \'' + uid + '\'?',
      actions: [
        {text: 'No'},
        {text: 'Yes', onAction: function(){
          $scope.loading = true;
          $http({
            method: 'POST',
            url: '/web/v1/account/delete',
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

  $scope.editGrants = function(acc){
    $rootScope.$broadcast('open-user-grants',{account: acc, cb: function(){
    }});
  }

  $scope.createAccount = function(){
    $rootScope.$broadcast('create-account',{cb: function(newUser){
      console.log("New user", newUser);

      $scope.loading = true;
      $http({
        method: 'POST',
        url: '/web/v1/account/create',
        data: newUser,
      }).then(function successCallback(response) {
        $scope.update();
      }, function errorCallback(response) {
        $scope.loading = false;
        $scope.error = response;
      });
    }});
  }

  $scope.update = function(){
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

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'accounts'){
      $scope.update();
    }
  });
}]);
