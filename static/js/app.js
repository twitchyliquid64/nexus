var app = angular.module('nexus', ['ui.materialize', 'angularMoment']);

app.controller('BodyController', ["$scope", "$rootScope", "$location", function ($scope, $rootScope, $location) {
    $scope.page = "home";
    $scope.refreshDash = function(){
      $scope.dashUpdated = Date.now();
    }
    $scope.changePage = function(pageName){
        $scope.page = pageName;
        $rootScope.$broadcast('page-change',{page: pageName});
        if ($scope.page == 'home')
          $scope.dashUpdated = Date.now();
        $location.hash(pageName);
    };

    $rootScope.$on('$locationChangeSuccess', function(event, args, args2) {
      console.log(location.hash);
      var spl = location.hash.split('##');
      if (spl.length > 1 && spl[1] != $scope.page){
        console.log("Doing URL-based change");
        $scope.changePage(spl[1]);
      }
   });
}]);

app.directive('loader', function($rootScope){
  return {
    scope: {
      loading: '=',
      error: '=',
    },
    //restrict E means its can only be used as an element.
    restrict: 'E',
    template: '<div class="progress" ng-show="loading"><div class="indeterminate"></div></div>  <blockquote ng-show="error"><h5>Error</h5>' +
        '<ul class="collection">' +
        '<li class="collection-item"><b>Error code</b>: {{ec()}}</li>' +
        '<li class="collection-item"><b>Explanation</b>: {{exp()}}</li>' +
        '<li class="collection-item"><b>The server said</b>: {{error.data}}{{error.reason}}</li>' +
        '</ul></blockquote>',
    link: function($scope, elem, attrs) {
      // scope = either parent scope or its own child scope if scope set.
      // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
    },
    controller: function($scope) {
      $scope.ec = function(){
        if (!$scope.error)return null;
        if ($scope.error.success === false)
          return 'N/A';
        return $scope.error.status;
      }
      $scope.exp = function(){
        if (!$scope.error)return null;
        if ($scope.error.status === -1)
          return "Network Error or server offline";
        if ($scope.error.success === false)
          return 'The server encountered a problem handling the request';
        return $scope.error.statusText;
      }
    },
  };
});



app.controller('AppController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.apps = [];
  $scope.update = function(){
    $scope.loading = true;
    $http({
      method: 'GET',
      url: '/apps/list'
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.apps = response.data;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'apps'){
      $scope.update();
    } else {
      $scope.apps = [];
    }
  });
}]);







app.directive('confirmationDialogModal', function($rootScope){
  return {
    //scope allows us to setup variable bindings to the parent scope. By default, we share the parent scope. For an isolated one, we should
    //pass an object as the scope attribute which has a dict of the variable name for us, and a string describing where and how to bind it.
    //scope: {color: '@colorAttr'} binds 'color' to the value of color-attr with a one way binding. Only strings supported here: color-attr="{{color}}"
    //scope: {color: '=color'} binds 'color' two way. Pass the object here: color="color".
    scope: {

    },
    //restrict E means its can only be used as an element.
    restrict: 'E',
    templateUrl: function(elem, attr){
      return "/static/views/confirmationDialogModal.html"
    },
    link: function($scope, elem, attrs) {
      // scope = either parent scope or its own child scope if scope set.
      // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
    },
    controller: function($scope) {
      $scope.open = false;
      $scope.title = 'Are you sure?';
      $scope.content = '';
      $scope.actions = [];

      $rootScope.$on('check-confirmation', function(event, args) {
        if (args.title)$scope.title = args.title;
        if (args.content)$scope.content = args.content;
        if (args.actions)$scope.actions = args.actions;
        $scope.open = true;
      });

      $scope.onPress = function(action){
        $scope.title = 'Are you sure?';
        $scope.content = '';
        $scope.open = false;
        if (action.onAction)action.onAction();
        $scope.actions = [];
      }
    }
  };
});
