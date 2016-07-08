(function() {
  'use strict';

  var dataset = $('#dataset');
  var workers = $('#workers');
  var title = $('#title');
  var results = $('#results');
  var loading = $('#loading');
  var chart = $('#scatter');
  var footer = {
    creditCard: $('#credit-card-footer'),
    iris: $('#iris-footer')
  };

  $('#submit').click(function() {
    var numWorkers = workers.val();
    if (!numWorkers) {
      return;
    }

    results.empty();
    chart.empty();
    loading.show();

    switch (dataset.val()) {
    case 'credit-card':
      title.text('Credit Card Defaults');
      $.get('/api/pca/credit-card/' + numWorkers, function(resp) {
        if (resp.status !== 'ok') {
          alert('Uh oh! ' + resp.message);
          return;
        }
        results.html('<p>Elapsed time: ' + resp.elapsed + ' seconds</p>');
        scatter('#scatter', '#loading', 'credit-card.csv', ['No Default', 'Default']);
        $('.footer').hide();
        footer.creditCard.show();
      });
      break;
    case 'iris':
      title.text('Iris');
      $.get('/api/pca/iris/' + numWorkers, function(resp) {
        if (resp.status !== 'ok') {
          console.log(resp);
          alert('Uh oh! ' + resp.message);
          return;
        }
        results.html('<p>Elapsed time: ' + resp.elapsed + ' seconds</p>');
        scatter('#scatter', '#loading', 'iris.csv', ['Setosa', 'Versicolor', 'Virginica']);
        $('.footer').hide();
        footer.iris.show();
      });
    default:
      break;
    }
  });
})();
