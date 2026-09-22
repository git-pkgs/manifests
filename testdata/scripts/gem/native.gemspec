Gem::Specification.new do |spec|
  spec.name = "native-widget"
  spec.extensions = [
    "ext/widget/extconf.rb",
    'ext/helper/extconf.rb',
  ]
  spec.extensions += %w[ext/extra/extconf.rb]
  spec.extensions << 'ext/final/extconf.rb'
  # spec.extensions << 'ext/comment/extconf.rb'
  spec.add_dependency "rake", ">= 0"
end
