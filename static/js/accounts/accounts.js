
app.controller('AccountViewController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.accounts = [];
  $scope.error = null;

  $scope.edit = function(user){
    console.log("edit", user);
    $rootScope.$broadcast('edit-account',{account: user, cb: function(newUser){
      console.log("Post-edit user", newUser);

      $scope.loading = true;
      $scope.error = null;
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
    $rootScope.$broadcast('accounts-auth-setup', {user: user});
    $scope.changePage('accounts-auth');
  }

  $scope.delete = function(uid){
    $rootScope.$broadcast('check-confirmation', {
      title: 'Confirm Deletion',
      content: 'Are you sure you want to delete user \'' + uid + '\'?',
      actions: [
        {text: 'No'},
        {text: 'Yes', onAction: function(){
          $scope.loading = true;
          $scope.error = null;
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
    $rootScope.$broadcast('open-user-grants',{account: acc, cb: function(d){
      switch (d.action) {
        case 'delete':
          $scope.loading = true;
          $scope.error = null;
          $http({
            method: 'POST',
            url: '/web/v1/account/delgrant',
            data: d,
          }).then(function successCallback(response) {
            $scope.update();
          }, function errorCallback(response) {
            $scope.loading = false;
            $scope.error = response;
          });
          break;
        case 'add':
          $scope.loading = true;
          $scope.error = null;
          $http({
            method: 'POST',
            url: '/web/v1/account/addgrant',
            data: d,
          }).then(function successCallback(response) {
            $scope.update();
          }, function errorCallback(response) {
            $scope.loading = false;
            $scope.error = response;
          });
          break;
      }
    }});
  }

  $scope.createAccount = function(){
    $rootScope.$broadcast('create-account',{cb: function(newUser){
      console.log("New user", newUser);

      $scope.loading = true;
      $scope.error = null;
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

  $scope.updateBuildInfo = function() {
    $http({
      method: 'GET',
      url: '/core/build'
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.buildInfo = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = null;
    $http({
      method: 'GET',
      url: '/web/v1/accounts'
    }).then(function successCallback(response) {
      $scope.accounts = response.data;
      $scope.updateBuildInfo();
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'accounts'){
      $scope.update();
    } else {
      $scope.accounts = [];
    }
  });
}]);
